# Sub2API Handover

## 2026-04-30 Commercial billing hardening

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- Implemented the first production-commerce hardening pass for the planned Stripe launch:
  - Subscription plans now have `stripe_price_id`, so a plan can map to a real recurring Stripe Price.
  - Stripe provider now supports recurring Checkout sessions for subscription plans and parses `checkout.session.completed`, `invoice.paid`, and `invoice.payment_failed` alongside existing PaymentIntent events.
  - Usage-log best-effort queue drops now fall back to synchronous writes instead of silently accepting missing analytics after billing succeeds.
  - Pricing status now exposes source label, source URL, full hash, fallback file, and whether the pricing source should be treated as authoritative.
  - Added ops reconciliation metric `usage_billing_missing_log_count` for alerting on rows that were billed through `usage_billing_dedup` but have no matching `usage_logs` row.
- Added migration `135_commercial_billing_hardening.sql` for `subscription_plans.stripe_price_id` and a usage-log reconciliation index.
- Added `docs/COMMERCIAL_BILLING.md` documenting the simple pricing policy:
  - top-up users: `0.70x`
  - subscription users: `0.50x`
  - internal/test users: separate non-revenue tracking.

### Verification

- Ran:
  - `go generate ./ent`
  - `go test ./internal/service`
  - `go test ./internal/repository -run 'TestUsageLogRepository|TestMigrations|TestOps'`
  - `go test ./...` from `backend`
  - `pnpm --dir frontend exec vue-tsc --noEmit --pretty false`
  - `pnpm --dir frontend run build`
- Result: passed. Frontend build still emits existing Vite chunk/dynamic-import warnings only.

## 2026-04-30 Stripe-only revenue/profit reporting

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- Changed admin money reporting so manually created/test-credit accounts no longer count as customer revenue or profit.
- New rule:
  - Traffic stats still count everyone: requests, tokens, active users, latency, raw usage logs.
  - Customer revenue/profit now only counts usage from non-admin, non-internal users/keys after that user has a successful Stripe-funded payment order.
  - Admin, explicitly internal, manually credited, test-credit, and pre-Stripe-payment usage now lands in the internal/test cost bucket instead of customer profit.
- Updated the admin dashboard labels and top token money snippets to show Stripe-funded charged usage and Stripe-funded upstream cost, not raw manual-credit usage.
- Updated the user spending ranking query to rank only Stripe-funded customer spend, so test users with manually assigned `$1000` balances do not appear as revenue leaders.
- Tightened the integration-test fixture helper so `internal_usage` is actually written when tests create users.

### Verification

- Ran:
  - `go test -tags=integration ./internal/repository -run TestUsageLogRepoSuite/TestDashboardProfitCountsOnlyStripeFundedUsage -count=1`
  - `go test -tags=unit ./internal/repository ./internal/handler/admin ./internal/service -run 'TestUsageLogRepository|TestDashboard|TestGetDashboard|TestPayment|TestAdminService_AdminUpdateAPIKeyInternalUsage' -count=1`
  - `go test ./...` from `backend`
  - `pnpm --dir frontend exec vue-tsc --noEmit`
  - `pnpm --dir frontend exec vitest run src/views/admin/__tests__/DashboardView.spec.ts src/components/charts/__tests__/ModelDistributionChart.spec.ts`
  - `pnpm --dir frontend build`
  - `git diff --check`
- Result: passed.
- Frontend build still emits existing Vite dynamic-import/chunk-size warnings and the existing Node `DEP0190` warning; these are warnings, not failures.

## 2026-04-30 GitHub Node 20 action warning cleanup

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- The non-blocking GitHub warning after the Sub2API deploy was not from the app's own `node-version: '20'`; it was from `pnpm/action-setup@v4`, whose JavaScript action runtime still used Node 20.
- Updated every `pnpm/action-setup` usage from `v4` to `v6` in:
  - `.github/workflows/backend-ci.yml`
  - `.github/workflows/security-scan.yml`
  - `.github/workflows/release.yml`
- Kept the project build Node version unchanged at `20` to avoid changing app/runtime assumptions in the same fix.
- Verification:
  - Commit `1d566736` passed GitHub CI run `25196057351`.
  - Commit `1d566736` passed GitHub Security Scan run `25196057344`.
  - The repeated Node 20 action annotation disappeared from the watched run output.

## 2026-04-30 Customer portal QA and provisioned-key rotation

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- Clarified product boundary:
  - Normal users log into the same website at `/login`.
  - After login, normal users land in the customer portal, not the admin console.
  - Admins still use `/admin/*`; non-admin users are redirected away from admin routes.
- Changed normal-user API key behavior:
  - Users no longer create, edit, disable, delete, choose groups, reset quotas, or change rate limits from the customer portal.
  - Users can list/view/copy their provisioned key and rotate the key secret from `/keys`.
  - Rotation keeps the same key record, usage history, group assignment, quota/limit settings, and internal/customer accounting flags; only the secret token value changes.
  - Backend customer routes now expose `GET /api/v1/keys`, `GET /api/v1/keys/:id`, and `POST /api/v1/keys/:id/rotate`; normal-user `POST /keys`, `PUT /keys/:id`, and `DELETE /keys/:id` were removed from the registered user routes.
- Customer-facing API docs now say the key is provisioned for the account and can be rotated, not self-created.
- Customer portal routing now allows normal users only into:
  - `/dashboard`
  - `/keys`
  - `/api-docs`
  - `/usage`
  - `/profile`
  - payment/order routes
  - configured custom pages
- Legacy normal-user pages like available channels and affiliate are redirected back to `/dashboard` for non-admins.
- Backend group/channel/monitor discovery endpoints now require admin role because those responses can expose routing/platform details.

### Verification

- Ran:
  - `go test -tags=unit ./internal/service -run 'TestApiKeyService_(Rotate|Delete)'`
  - `go test -tags=unit ./internal/server -run TestAPIContracts`
  - `go test -tags=unit ./internal/service ./internal/server ./internal/handler ./internal/repository`
  - `pnpm --dir frontend exec vue-tsc --noEmit`
  - `pnpm --dir frontend exec vitest run src/router/__tests__/guards.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts src/router/__tests__/title.spec.ts`
  - `pnpm --dir frontend build`
  - `git diff --check`
- Result: passed
- Build still shows existing Vite chunk/dynamic-import warnings and the existing Node `DEP0190` warning; these are warnings, not failures.

## 2026-04-30 Admin API docs and simplified user portal

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- Added a customer-safe API docs page rendered inside the app:
  - user route: `/api-docs`
  - admin route: `/admin/api-docs`
  - content source: `frontend/src/content/sub2apiUserApiDoc.ts`
- Added an admin dashboard launch-docs card linking to `/admin/api-docs`.
- Added an admin sidebar item for API Docs.
- Simplified the regular user sidebar toward the launch flow:
  - Dashboard
  - API Keys
  - API Docs
  - Usage
  - Recharge / Subscription and Orders only when payment is enabled
  - Profile
- Removed nonessential user-sidebar entries from the default customer view, so normal users do not see the admin-heavy operational surface.
- Admin-only routes remain protected by `requiresAdmin: true`; non-admin users are redirected away from `/admin/*`.
- Hardened i18n startup for test/runtime environments where `localStorage.getItem` is missing.

### Verification

- Ran:
  - `pnpm --dir frontend exec vue-tsc --noEmit`
  - `pnpm --dir frontend exec vitest run src/components/layout/__tests__/AppSidebar.spec.ts src/router/__tests__/title.spec.ts`
  - `pnpm --dir frontend build`
  - `git diff --check`
- Result: passed

## 2026-04-30 Upstream v0.1.121 update compatibility pass

- Scope: `/Users/benzhang/dev/aptidus-sub2api`.
- The official upstream repo was fetched as `upstream=https://github.com/Wei-Shaw/sub2api.git`; latest tag checked was `v0.1.121`.
- A normal Git merge is unsafe for this checkout because the local Aptidus repo and official upstream have unrelated histories. The safe path used here was patch-based: `v0.1.119..v0.1.121` was applied in a throwaway worktree first, conflicts were resolved there, then the tested result was copied into the real checkout.
- Rollback backups were created before applying to the real checkout:
  - `/tmp/sub2api-real-before-0.1.121-tracked.patch`
  - `/tmp/sub2api-real-before-0.1.121-untracked.tgz`
- Preserved Aptidus-local work while bringing in the upstream fixes:
  - user/API-key internal usage accounting stayed in place
  - user API-doc/dashboard changes stayed in place
  - admin API-key group/internal-usage behavior stayed in place
  - upstream API-key rate-limit reset behavior was added alongside it
  - upstream Anthropic OAuth mimicry behavior stayed compatible with non-Claude-Code user agents
- Updated `backend/cmd/server/VERSION` to `0.1.121`.
- Added a Vitest-only storage setup for Node 25, because this machine exposes a broken global `localStorage`; production code is unchanged by that test setup.
- Fixed low-risk frontend regressions uncovered during the full suite:
  - chart/dashboard cost formatters now tolerate missing cost fields instead of throwing
  - OpenAI OAuth usage refresh clears cache on account row updates
  - stale legacy table page-size localStorage markers no longer override server defaults

### Verification

- Ran:
  - `go test ./...`
  - `go test ./internal/handler/admin ./internal/service ./internal/pkg/apicompat ./internal/pkg/httputil ./internal/repository ./internal/server`
  - `pnpm --dir frontend test:run`
  - `pnpm --dir frontend build`
  - `git diff --check`
- Result: passed.

## 2026-04-30 Profit dashboard source of truth

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- The payment dashboard was kept focused on cash/order reporting only. It should not run a second usage-profit query.
- The main admin dashboard is now the usage-profit source of truth because it already receives the existing usage accounting fields:
  - charged usage USD: `actual_cost`
  - estimated upstream account cost USD: `account_stats_cost`
  - standard/list-model cost USD: `total_cost`
- Added two admin dashboard cards:
  - Today Usage Profit = today customer charged usage minus today customer upstream account cost
  - Total Usage Profit = total customer charged usage minus total customer upstream account cost
- Added an Internal Usage Cost card. Admin-role usage is tracked as internal cost and excluded from customer profit, so local SpearAgent/project usage does not look like launch revenue if it runs under an admin user.
- Added first-class `internal_usage` flags on `users` and `api_keys`:
  - `users.internal_usage = true` makes every API call from that user count as internal cost, not customer profit.
  - `api_keys.internal_usage = true` makes only that key count as internal cost, not customer profit.
  - `users.role = 'admin'` is still automatically treated as internal.
  - Admin UI can mark a user as internal, create an internal key for a user, and flip an existing key between internal/customer tracking.
- Migration added: `backend/migrations/134_add_internal_usage_flags.sql`.
- Stripe fees are intentionally excluded per user request. Current gross usage profit is model usage revenue minus upstream account cost.
- This avoids duplicate payment accounting work: cost/rate/token calculation stays in the existing usage stats path, while the payment dashboard remains cash/order reporting.
- Amount precision validation rejects fractional cents while allowing harmless trailing zeros such as `1.230` because that still equals exactly 123 cents.
- While running the full service unit package, fixed a nil-load-balancer panic in webhook provider fallback. Multiple Stripe instances still use enabled instance candidates; single-instance legacy fallback can still use the registry provider.

### Verification

- Ran:
  - `pnpm --dir frontend exec vue-tsc --noEmit`
  - `go test -tags=unit ./internal/payment ./internal/service -run 'TestYuanToFen|TestValidateYuanAmountFloat|TestCalculatePayAmount|TestValidateProviderConfig_StripeRequiresAllRuntimeKeys|TestGetWebhookProvidersReturnsAllEnabledStripeInstances|TestCalculateStatsCost_TokenBilling_UsesExactMoneyRounding'`
  - `go test ./internal/payment/provider`
  - `go test -tags=unit ./internal/repository ./internal/handler/admin -run 'TestUsageLogRepository|TestDashboard|TestGetDashboard'`
  - `go test -tags=unit ./internal/repository ./internal/handler/admin -run 'TestUsageLogRepository|TestDashboard|TestGetDashboard|TestAdminAPIKey|TestAdminUser'`
  - `go test -tags=unit ./internal/repository`
  - `go test -tags=unit ./internal/handler/admin`
  - `go test -tags=unit ./internal/service -run 'TestAdminService_AdminUpdateAPIKeyGroupID|TestAdminService_AdminUpdateAPIKeyInternalUsage|TestYuanToFen|TestValidateYuanAmountFloat|TestCalculateStatsCost'`
  - `go test -tags=unit ./internal/service -run 'TestPayment|TestCalculateStatsCost|TestResolveAccountStatsCost'`
  - `go test -tags=unit ./internal/service`
  - `git diff --check`
- Result: passed

## 2026-04-30 Upstream v0.1.121 second-pass review

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- Continued the safety review after applying upstream `v0.1.121`.
- Main compatibility point checked: Anthropic OAuth accounts still run Claude Code mimicry for non-Claude-Code clients across native Anthropic messages, OpenAI Chat Completions bridge, OpenAI Responses bridge, and count-tokens requests. That preserves the local requirement that authorized Anthropic upstream accounts can serve all supported user-agent styles.
- Found and fixed one hardening issue in `backend/internal/service/gateway_service.go`: sticky-session diagnostic logs had been added at normal `Info` level and could include raw metadata/session values. They are now debug-level only and use short/redacted session identifiers.
- Filename-only credential scan found no live-looking `nvapi-`, `tp-`, or long `sk-...` secrets in the searched docs/backend/frontend tree. Matches were placeholders/env names/test keys only.
- Verification after the hardening edit:
  - `go test ./...` from `backend`: passed.
  - `pnpm --dir frontend test:run`: passed, 91 files / 545 tests.
  - `pnpm --dir frontend build`: passed, with existing Vite chunk/dynamic-import warnings only.
  - `git diff --check`: passed.

## 2026-04-30 Claude Code `[1m]` model suffix fix

- Scope: `/Users/benzhang/dev/aptidus-sub2api`.
- User report: Leif's active Sub2API key failed in Claude Code with `claude-sonnet-4-6[1m]` and the client said the selected model might not exist.
- Live production repro before the fix:
  - `POST /v1/messages` with `model=claude-sonnet-4-6[1m]` returned HTTP 502 upstream error.
  - The same key and same request with `model=claude-sonnet-4-6` returned HTTP 200 and `ok`.
- Root cause: Claude/Cowork can show/send `[1m]` as a long-context display suffix, but Sub2API forwarded that literal string to Anthropic instead of stripping it to the real upstream model id. Auth and the user key were not the root problem.
- Fix:
  - Added Claude model display-suffix stripping in `backend/internal/pkg/claude/constants.go`.
  - Made Anthropic account model-mapping lookup normalize suffixed IDs before whitelist/mapping checks.
  - Applied the same normalization to Anthropic API-key passthrough, `count_tokens`, and OpenAI-compatible Anthropic bridge paths.
  - Follow-up after live QA: pricing lookup also normalizes the suffix before cost lookup, so `claude-sonnet-4-6[1m]` does not fuzzy-match an older Sonnet pricing row when a normalized model price exists.
  - Added regression coverage for `claude-sonnet-4-6[1m]` normalization and Anthropic API-key passthrough.
- Verification passed:
  - `go test ./internal/pkg/claude`
  - `go test ./internal/service -run 'TestAccount(GetMappedModel|ResolveMappedModel)|TestGatewayService_AnthropicAPIKeyPassthrough_ModelMappingEdgeCases'`
  - `go test ./internal/service -run 'TestGetModelPricing_NormalizesClaudeDisplaySuffixBeforeLookup|TestAccount(GetMappedModel|ResolveMappedModel)|TestGatewayService_AnthropicAPIKeyPassthrough_ModelMappingEdgeCases'`
  - `go test ./...`
  - `pnpm --dir frontend exec vue-tsc --noEmit --pretty false`
  - `pnpm --dir frontend run build` passed with existing Vite dynamic-import/chunk-size warnings only.
- Deployment:
  - Commit `febb5c42` was pushed to `aptidus/sub2api` `main`; Railway deployment `140f35ce-b643-4f32-9467-dc9c1f00c6bb` succeeded.
  - Live QA after `febb5c42`: `POST /v1/messages` with `model=claude-sonnet-4-6[1m]` returned HTTP 200 and `ok`.
  - A second follow-up commit is required for the pricing-normalization log issue discovered during live QA.

## 2026-04-30 Profitability setup inputs needed

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- The repo already has token/cost tracking, provider-instance pinning, and OAuth usage fetching for accounts that can query usage.
- The next blocker is business data: exact upstream cost, quota rules, and which upstream accounts/models belong to which pool.

### Needed from user

- One row per upstream account with:
  - account name or ID
  - auth type: Anthropic OAuth, Setup Token, or API key
  - monthly fixed cost
  - 5-hour quota rule
  - weekly quota rule
  - whether the quota is shared across all models or model-specific
  - which models are allowed on that account
  - whether the account is dedicated or part of a shared pool
  - any overage, throttling, or manual reset rules
- The target margin or markup rule, so profitability can be checked automatically.
- Confirmation of whether Anthropic OAuth usage should be treated as the source of truth when it is available, with local token logs as fallback.

### Current code facts

- `AccountUsageService.GetUsage` can fetch usage for OAuth accounts with the right scope; Setup Token accounts only get an estimated 5-hour window and no 7-day usage.
- Payment/accounting paths now use decimal math instead of plain float math for the main money calculations.
- Stripe provider validation now requires all runtime keys and multi-instance Stripe webhook lookup is supported.

## 2026-04-30 Stripe readiness review

- Scope: `/Users/benzhang/dev/aptidus-sub2api`
- This run fixed the Stripe launch blockers and tightened money math.
- Target question: whether the current payment stack is ready for Stripe launch and whether profit/cost math is exact enough for production accounting.

### Findings

- The Stripe amount conversion path now rejects over-precision amounts instead of silently truncating them.
- Stripe provider validation now requires `secretKey`, `publishableKey`, and `webhookSecret` before an instance can be enabled.
- Stripe webhooks now try every enabled Stripe instance, so multiple Stripe secrets no longer make callbacks ambiguous.
- Payment totals and account stats now sum with decimal math before converting back to floats for the API response, which removes the main rounding drift in the reporting path.

### Verification

- Ran:
  - `go test -tags=unit ./internal/payment ./internal/service -run 'TestYuanToFen|TestFenToYuan|TestYuanToFenRoundTrip|TestValidateYuanAmountFloat|TestCalculatePayAmount|TestValidateProviderConfig_StripeRequiresAllRuntimeKeys|TestGetWebhookProvidersReturnsAllEnabledStripeInstances|TestCalculateStatsCost_TokenBilling_UsesExactMoneyRounding'`
  - `go test ./internal/payment/provider`
- Result: passed

## 2026-04-29 Responses tool-argument buffering fix deployed

- Scope: `/Users/benzhang/dev/aptidus-sub2api`.
- Fixed the non-streaming `/v1/responses` bridge for Anthropic streaming tool calls. Anthropic starts a `tool_use` block with placeholder `input:{}` and then streams the real tool arguments as `input_json_delta`; the buffered Responses converter was concatenating these into malformed `{}{...}` arguments.
- `appendRawJSON` now replaces an empty `{}` placeholder with the first real streamed JSON fragment instead of appending to it.
- Added regression coverage in `backend/internal/service/gateway_forward_as_responses_test.go` for this exact buffered tool-call path.
- Verification passed:
  - `go test -tags=unit ./internal/service -run 'TestHandleResponsesBufferedStreamingResponse_ReplacesToolUsePlaceholderInput|TestHandleResponsesBufferedStreamingResponse_PreservesMessageStartCacheUsage|TestResolveGatewayGroup_AllowsResponsesBridgeForClaudeCodeOnlyGroup'`
  - `go test ./internal/pkg/apicompat`
- Commit `d3beb62` was pushed to `aptidus/sub2api` `main`; Railway production service `sub2api-app` deployed it successfully.
- Live production QA passed against `https://sub2api-app-production.up.railway.app/v1/responses` using the local Sub2API key:
  - plain `claude-opus-4-7` Responses request returned `ok`
  - tool-call request returned a valid Responses `function_call` with clean JSON arguments
  - tool-output continuation request translated back through Claude and returned the expected answer

## 2026-04-29 SpearAgent Codex Responses bridge for Claude Code-only groups

- Scope: `/Users/benzhang/dev/aptidus-sub2api`.
- Fixed the Anthropic `/v1/responses` bridge so SpearAgent Codex requests can use Claude models through Sub2API.
- Removed the handler-level assumption that `/v1/responses` can never be Claude-Code-compatible. The handler now marks Responses bridge requests as Claude-Code-compatible before account selection, so `claude_code_only` groups can select their Claude Code accounts.
- The actual upstream shape remains server-side in `ForwardAsResponses`: Codex/OpenAI Responses payloads are converted to Anthropic `/v1/messages`, Claude Code mimic headers/beta headers are applied for OAuth accounts, the TLS fingerprint transport is used, and Anthropic streaming is converted back to Responses events.
- Added regression tests for:
  - `claude_code_only` group access when the Responses bridge marks the request as Claude-Code-compatible.
  - Codex-style Responses tool calls/tool results and `xhigh` reasoning translating to Anthropic tool blocks plus `output_config.effort=max`.
- Verification passed:
  - `go test ./internal/pkg/apicompat`
  - `go test -tags=unit ./internal/service -run 'TestResolveGatewayGroup_AllowsResponsesBridgeForClaudeCodeOnlyGroup|TestSelectAccountWithLoadAwareness_UsesFallbackGroupForChannelRestriction|TestSelectAccountForModelWithExclusions_UsesFallbackGroupForChannelRestriction'`
  - `go test ./internal/handler`
- Deployment note: Railway status showed the CLI linked to `sub2api-cn-relay`, while the main production service `sub2api-app` is GitHub-backed from `aptidus/sub2api` `main`. Push to `main` should trigger the main app deployment; do not run `railway up` from the current link without switching service.
