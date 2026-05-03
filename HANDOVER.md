# Sub2API Handover

## 2026-05-02 Leif 503 compatibility + user model visibility

- Scope: `/Users/benzhang/dev/aptidus-sub2api`.
- User reported that Leif's Claude Code client returned:
  - `503 {"error":{"message":"No available accounts: no available accounts","type":"api_error"},"type":"error"}`
  - Client label showed `Opus 4.6 (1M context)`.
- Root cause:
  - Production was intentionally changed to advertise/route latest Claude models only.
  - The older client route `claude-opus-4-6[1m]` normalized the `[1m]` display suffix, but still requested stale `claude-opus-4-6`.
  - Current production account mappings only contain `claude-opus-4-7`, so scheduler filtering found no eligible Anthropic account.
- Backend fix:
  - Added a compatibility alias in `backend/internal/pkg/claude/constants.go` so `claude-opus-4-6` and `claude-opus-4-6[1m]` route to `claude-opus-4-7`.
  - Added coverage in `backend/internal/pkg/claude/constants_test.go`.
  - Added coverage in `backend/internal/service/account_wildcard_test.go` proving an Anthropic account with only `claude-opus-4-7` mapping can serve stale `claude-opus-4-6[1m]` client requests.
  - `/v1/models` remains latest-only because this alias is routing compatibility, not a public model-list rollback.
- User portal fix:
  - Added a read-only `Models` action to the normal user's `/keys` table in `frontend/src/views/user/KeysView.vue`.
  - The modal calls `/v1/models` with that exact API key as `Authorization: Bearer ...`, so users see the same model list their key can actually use.
  - This avoids exposing admin upstream accounts, groups, quotas, provider secrets, or internal scheduler details.
  - Added English and Chinese copy in `frontend/src/i18n/locales/en.ts` and `frontend/src/i18n/locales/zh.ts`.
- Verification:
  - `pnpm --dir frontend exec vue-tsc --noEmit`
  - `pnpm --dir frontend build`
  - `git diff --check`
  - `go test ./internal/pkg/claude ./internal/service -run 'TestNormalizeModelIDStripsClaudeDisplaySuffix|TestGatewayService|TestGetModelPricing_NormalizesClaudeDisplaySuffixBeforeLookup'`
  - `go test -tags unit ./internal/service -run 'TestAccount(GetMappedModel|ResolveMappedModel)' -v`
- Deployment/live verification:
  - Not deployed in this turn.
  - After deploy, verify Leif's key/client against `claude-opus-4-6[1m]` and confirm the user `/keys -> Models` modal returns only the latest approved public models.
- No API keys, OAuth tokens, admin credentials, customer secrets, or proxy credentials were written to this handover.

## 2026-05-02 Anthropic OAuth TLS default

- Scope: `/Users/benzhang/dev/aptidus-sub2api` production.
- User asked to keep the shared Oxylabs IP path, but make sure TLS fingerprinting is enabled for all Anthropic OAuth accounts and future Anthropic OAuth/setup-token accounts. Codex/OpenAI accounts should not use this Anthropic TLS feature.
- Code change:
  - Added `EnsureAnthropicOAuthTLSFingerprintEnabled` in `backend/internal/service/account.go`.
  - Applied it in the admin create/update path, regular account create/update path, and CRS sync create/update path.
  - Behavior: if an account is `platform=anthropic` and `type=oauth` or `type=setup-token`, Sub2API forces `extra.enable_tls_fingerprint=true`.
  - Behavior: `platform=openai` / Codex OAuth accounts and Anthropic API-key accounts are left untouched.
- Tests passed:
  - `go test ./internal/service -run 'TestEnsureAnthropicOAuthTLSFingerprintEnabled|TestAccount_IsAnthropicAPIKeyPassthroughEnabled'`
  - `go test ./internal/service ./internal/handler/admin -run 'TestEnsureAnthropicOAuthTLSFingerprintEnabled|TestAccount|TestCRS|TestGatewayService'`
- Deployment:
  - Railway production deployment `efa9254d-5bbb-4c1d-86df-2c8a23e4adc9` succeeded.
- Live verification:
  - Production admin account list has 12 upstream accounts.
  - Anthropic accounts `3`, `6`, `7`, `8`, `9`, `10` all show `enable_tls_fingerprint=true`.
  - OpenAI/Codex accounts `5`, `11`, `12`, `13`, `14`, `15` show no TLS fingerprint flag, as intended.
- No OAuth tokens, admin credentials, customer API keys, or proxy credentials were written to this handover.

## 2026-05-02 Spear Proxy OAuth import

- Scope: `/Users/benzhang/dev/aptidus-sub2api` production plus local Spear Proxy account store at `/Users/benzhang/.anti-api/auth`.
- Copied the usable local Spear Proxy Codex/OpenAI OAuth account for `tianyiz2020@gmail.com` into Sub2API production.
- Created production upstream account:
  - id `5`
  - name `SpearProxy Codex - tianyiz2020@gmail.com`
  - platform/type `openai` / `oauth`
  - proxy id `1` (`Oxylabs US ISP 8003`)
  - concurrency `4`
  - WS mode `off`
  - privacy mode `training_off`
- Verified with the built-in admin account test: `POST /api/v1/admin/accounts/5/test` using `gpt-5.4` returned `test_complete` / `success: true`.
- Did not import the local `ben.zhang.22@gmail.com` Codex row because it had no refresh token and its access token was already expired; reauthenticate that account first if it should become a Sub2API upstream account.
- No OAuth tokens, admin credentials, or proxy credentials were written to this handover.

## 2026-05-02 Upstream latest-model-only whitelist

- Scope: `/Users/benzhang/dev/aptidus-sub2api` production.
- User asked to keep only the latest model/image/mini versions for Sub2API upstream accounts and drop older versions.
- Updated production account `credentials.model_mapping`:
  - account id `3` (`anthropic` / `setup-token`) allows only `claude-opus-4-7`, `claude-sonnet-4-6`, and `claude-haiku-4-5-20251001`.
  - account id `5` (`openai` / `oauth`) allows only `gpt-5.5`, `gpt-5.4-mini`, and `gpt-image-2`.
- Verification:
  - `/v1/models` with the current customer key returns only the three latest Claude family models.
  - `claude-opus-4-6` now returns no available account, proving older Claude Opus is no longer schedulable.
  - `claude-sonnet-4-6` still returns HTTP `200`.
- Caveat: the OpenAI account whitelist is set, but the current customer key is still attached to the Anthropic `default` group, so OpenAI models are not visible from that key until an OpenAI-facing group/key/channel path is configured.
- No secrets were written to this handover.

## 2026-05-01 Official upstream 0.1.121 review

- Scope: `/Users/benzhang/dev/aptidus-sub2api`.
- Checked official upstream `Wei-Shaw/sub2api`:
  - official `upstream/main` is at `0.1.121`
  - local fork already had almost all `v0.1.119 -> v0.1.121` upstream code from earlier sync work
  - a blind merge is unsafe because this fork was imported as a deploy snapshot and does not share normal Git ancestry with upstream; direct `HEAD..upstream/main` looks like thousands of deletes that would remove our custom commercial/auth/API-doc changes.
- Applied the upstream delta safely as a patch and resolved the real differences:
  - preserved our redacted/debug sticky-session logs so raw metadata/session IDs are not written at info level
  - preserved our Anthropic model normalization behavior that fixed Claude alias / `[1m]` routing compatibility
  - adopted upstream's restored table page-size persistence behavior and updated the stale local test expectation
- Net code diff after resolution is intentionally small:
  - `frontend/src/composables/usePersistedPageSize.ts`
  - `frontend/src/composables/__tests__/usePersistedPageSize.spec.ts`
  - `backend/internal/service/gateway_service.go` ordering only around model mapping

### Verification

- Ran:
  - `go test ./internal/service ./internal/handler/admin ./internal/pkg/apicompat ./internal/pkg/httputil`
  - `go test ./...` from `backend`
  - `pnpm --dir frontend exec vitest run`
  - `go build ./cmd/server`
  - `pnpm --dir frontend build`
  - `git diff --check`
- Result: passed. Frontend build still emits existing Vite dynamic-import/chunk-size warnings only.

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

## 2026-04-30 Hugo admin promotion

- Scope: `/Users/benzhang/dev/aptidus-sub2api`, Railway production service `sub2api-app`.
- User requested `hugochougt@gmail.com` be made an admin.
- Production DB was updated from inside the Railway app container because local access cannot resolve `postgres.railway.internal`.
- Result:
  - User row `id=3`, `hugochougt@gmail.com` is now `role=admin`, `internal_usage=true`, `status=active`.
  - Duplicate user row `id=6`, `hugochougt@gmail.com` is also now `role=admin`, `internal_usage=true`, `status=active`.
- Note: because this was a direct DB update, any already-open Hugo dashboard session may need logout/login to pick up the admin role immediately.

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
  - Follow-up commit `ad5c0fcf` was pushed; Railway deployment `371e707d-79cc-45ca-8b52-c3828bbfc7b5` succeeded.
  - Final live QA after `ad5c0fcf`: `POST /v1/messages` with `model=claude-sonnet-4-6[1m]` returned HTTP 200 and `ok`; recent logs showed the request completed with status 200 and no `[Pricing] Fuzzy matched claude-sonnet-4-6[1m]` line.

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

## 2026-05-02 Spear Proxy OAuth migration into default group

- Scope: production Sub2API at `https://sub2api-app-production.up.railway.app` plus production Spear Proxy `anti-api-proxy-production`.
- Imported production Spear Proxy OAuth auth files from Railway volume `/data/auth`:
  - 5 Anthropic OAuth accounts: `alex@aiama.xyz`, `ben@aiama.xyz`, `dave@aiama.xyz`, `ed@aiama.xyz`, `spear@aiama.xyz`
  - 5 Codex/OpenAI OAuth accounts: `ben.zhang.22@gmail.com`, `benzhang@alumni.ucla.edu`, `benzhang0819@gmail.com`, `benzhang.ys@gmail.com`, `spear@aiama.xyz`
- Sub2API now has 12 active upstream accounts in group `default`:
  - existing accounts `3` and `5`
  - newly imported accounts `6` through `15`
- All 12 accounts use proxy id `1` (`Oxylabs US ISP 8003`) and are bound to `default`.
- Latest-only model mappings:
  - Claude/Anthropic accounts: `claude-opus-4-7`, `claude-sonnet-4-6`, `claude-haiku-4-5-20251001`
  - Codex/OpenAI accounts: `gpt-5.5`, `gpt-5.4-mini`, `gpt-image-2`
- Code change:
  - Updated `backend/internal/server/routes/gateway.go` so a mixed `default` group can route `gpt-*` requests to the OpenAI gateway by inspecting the request `model` field, while Claude requests keep using the Anthropic gateway.
  - Added regression coverage in `backend/internal/server/routes/gateway_test.go`.
- Verification:
  - `go test ./internal/server/routes ./internal/handler -run 'TestGatewayRoutes|TestShouldUseOpenAIGateway|TestGatewayHandler'`
  - Railway deployment `dfe60069-7cfd-4b3d-8028-474ddb86ed16` for `sub2api-app` succeeded.
  - Customer `/v1/models` returns exactly: `claude-haiku-4-5-20251001`, `claude-opus-4-7`, `claude-sonnet-4-6`, `gpt-5.4-mini`, `gpt-5.5`, `gpt-image-2`.
  - `claude-sonnet-4-6` live request returned HTTP `200` and `ok`.
  - `gpt-5.5` live `/v1/responses` request returned HTTP `200`, status `completed`, and output text `ok`.
  - old `claude-opus-4-6` returned HTTP `503`.
  - old `gpt-5.2` returned HTTP `503`.
  - Direct account tests passed for imported account id `6` (`claude-sonnet-4-6`) and imported account id `11` (`gpt-5.5`).
- Oxylabs/IP verification:
  - Spear Proxy selected endpoint is `disp.oxylabs.io:8003`.
  - A live egress probe from Spear Proxy through that endpoint observed IP `9.142.112.188`.
  - Sub2API proxy id `1` is also `disp.oxylabs.io:8003` and reports observed IP `9.142.112.188`.
  - This means the upstream-visible IP stayed the same for the migration/authentication path.
- Note: the user mentioned 14 Spear Proxy upstream accounts. The production OAuth auth store contained 10 OAuth files; the remaining count appears to come from non-OAuth/env/session entries in quota/cache state, not importable OAuth account files.
- No OAuth access token, refresh token, admin password, or proxy password was written to this handover.

## 2026-05-02 Final migration QA and Spear Proxy shutdown

- Scope: production Sub2API `sub2api-app` and production Spear Proxy `anti-api-proxy`.
- Account-type clarification for `tianyiz2020@gmail.com`:
  - account id `5`: OpenAI/Codex OAuth account (`platform=openai`, `type=oauth`).
  - account id `3`: Claude setup-token account (`platform=anthropic`, `type=setup-token`), which may appear token-like in the UI but is not a manually entered API key.
  - No queried Tianyi upstream row had `type=apikey`.
- Account model mapping cleanup:
  - Initial all-account/all-latest-model QA found account id `15` could run `gpt-5.5` and `gpt-5.4-mini` but not `gpt-image-2`; upstream returned `Tool choice 'image_generation' not found in 'tools' parameter.`
  - Removed `gpt-image-2` from account id `15` `credentials.model_mapping`.
  - Account id `15` remains active for latest GPT text and mini models.
- Final advertised-account QA:
  - Tested every active migrated account against every model it advertises.
  - Result: `35` total checks, `35` passed.
  - Claude account ids `3`, `6`, `7`, `8`, `9`, and `10` passed `claude-opus-4-7`, `claude-sonnet-4-6`, and `claude-haiku-4-5-20251001`.
  - OpenAI account ids `5`, `11`, `12`, `13`, and `14` passed `gpt-5.5`, `gpt-5.4-mini`, and `gpt-image-2`.
  - OpenAI account id `15` passed `gpt-5.5` and `gpt-5.4-mini`.
- Final customer-facing QA:
  - `/v1/models` returns exactly: `claude-haiku-4-5-20251001`, `claude-opus-4-7`, `claude-sonnet-4-6`, `gpt-5.4-mini`, `gpt-5.5`, `gpt-image-2`.
  - Claude latest models returned HTTP `200` and `ok` through `/v1/messages`.
  - GPT latest text models returned HTTP `200`, status `completed`, and `ok` through `/v1/responses`.
  - `gpt-image-2` returned HTTP `200` with image data through `/v1/images/generations`.
  - Old `claude-opus-4-6` and old `gpt-5.2` returned HTTP `503`, confirming older models are no longer user-facing.
- Spear Proxy was disabled after QA:
  - Set production Spear Proxy `/data/settings.json` `proxyEnabled=false`.
  - Authenticated Spear Proxy API verification returned HTTP `503` with `Proxy is currently disabled. Enable it from the dashboard.`
  - This prevents old Spear Proxy API endpoint users from sending traffic to the migrated upstream accounts.
- No OAuth access token, refresh token, admin password, customer API key, or Spear Proxy API key was written to this handover.

## 2026-05-02 Account hardening, compact probes, and count_tokens fix

- Scope: production Sub2API `sub2api-app`.
- OpenAI compact probes:
  - The `Compact 未知` badge in the admin UI meant no explicit compact probe had been persisted.
  - Ran compact tests for OpenAI account ids `5`, `11`, `12`, `13`, `14`, and `15`.
  - All six passed and now show `openai_compact_supported=true`.
- Anthropic hardening:
  - Before this pass, imported Claude OAuth account ids `6` through `10` lacked the stronger flags already present on account id `3`.
  - Updated account ids `6`, `7`, `8`, `9`, and `10`:
    - `enable_tls_fingerprint=true`
    - `session_id_masking_enabled=true`
    - `session_idle_timeout_minutes=5`
    - `window_cost_sticky_reserve=10`
    - `base_rpm=15`
    - `rpm_strategy=tiered`
    - `user_msg_queue_mode=throttle`
  - Post-update account QA passed for all Claude account/model combinations: `18/18`.
- Capacity:
  - Live group capacity after hardening: `concurrency_max=48`, `concurrency_used=0`.
  - Platform split from ops concurrency: Anthropic `24`, OpenAI `24`.
  - All 12 accounts are active/schedulable with `concurrency=4`.
  - Claude group RPM guard is now `90` max (`15` per Claude account).
- Count-token repair:
  - Live `/v1/messages/count_tokens` initially failed because Claude OAuth mimicry added generation-only fields (`max_tokens`, then `temperature`) that Anthropic rejects on the count-token endpoint.
  - Updated `backend/internal/service/gateway_service.go` so count_tokens strips generation-only fields before forwarding.
  - Updated tests in `backend/internal/service/gateway_anthropic_apikey_passthrough_test.go`.
  - Tests passed:
    - `go test ./internal/service -run 'TestGatewayService_(CountTokensOAuthMimicryStripsGeneratedMaxTokens|AnthropicAPIKeyPassthrough_ModelMappingPreservesOtherFields|AnthropicAPIKeyPassthrough_ForwardCountTokensPreservesBody|AnthropicAPIKeyPassthrough_CountTokens404PassthroughNotError)'`
    - `go test ./internal/service ./internal/handler ./internal/server/routes -run 'TestGatewayService|TestGatewayHandler|TestGatewayRoutes|TestShouldUseOpenAIGateway|TestCountTokens|TestIsCountTokensUnsupported404'`
  - Railway deployment `3cc7cf7c-585e-424f-b72f-1065c2fbd864` succeeded.
  - Production verification:
    - `/v1/messages/count_tokens` returned HTTP `200`, `input_tokens=12`, even with client-sent `max_tokens` and `temperature`.
    - `/v1/messages` with `claude-sonnet-4-6` returned HTTP `200` and `ok`.
    - `/v1/responses` with `gpt-5.5` returned HTTP `200`, status `completed`, and `ok`.
    - Fresh usage logs were written with nonzero tokens and token billing costs for both Claude and GPT requests.
- No OAuth access token, refresh token, admin password, customer API key, or proxy password was written to this handover.

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
