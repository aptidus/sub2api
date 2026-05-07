# Sub2API Handover

## 2026-05-06 SpearRelay removal and default WebUI rollback

- Scope: `/Users/benzhang/dev/aptidus-sub2api`.
- User decided not to use the separate SpearRelay frontend and to keep using the default Sub2API WebUI only.
- Code changes made:
  - Deleted the standalone `spearrelay/` static frontend.
  - Removed the Docker image copy step for `spearrelay/`.
  - Removed backend `/spearrelay` static-file routes from `backend/internal/server/routes/common.go`.
  - Removed the `/spearrelay` embedded-frontend bypass from `backend/internal/web/embed_on.go`.
  - Removed the dedicated `/api/v1/customer/...` route registration and deleted `backend/internal/server/routes/customer.go`.
  - Removed customer-portal-only auth helper methods from `backend/internal/handler/auth_handler.go`; the default auth routes remain.
- Operational intent:
  - Normal access should go through the default Sub2API WebUI.
  - No Stripe/payment setting was enabled in this rollback.
  - Existing admin/operator functionality should remain intact.
- Local verification:
  - `gofmt` passed on edited Go files.
  - `git diff --check` passed.
  - `rg -n "SpearRelay|spearrelay|CustomerLogin|CustomerRefreshToken|RegisterCustomerRoutes|/api/v1/customer" backend frontend Dockerfile` returned no active-code matches.
  - `go test ./internal/server/routes ./internal/server ./internal/handler` passed.
  - `go test ./internal/service ./internal/repository ./internal/handler ./internal/handler/admin ./internal/server ./internal/server/routes` passed.
  - `npm run typecheck` in `frontend/` passed.
- Pending production verification after push:
  - Verify `/health` stays healthy.
  - Verify `/` serves the default Sub2API WebUI.
  - Verify `/spearrelay/` no longer serves the removed frontend.
  - Verify `/api/v1/customer/auth/login` no longer exists.
- No OAuth access token, refresh token, admin key, customer API key, database password, or Stripe secret was written to this handover.

## 2026-05-06 SpearRelay customer portal backend connection

- Scope: `/Users/benzhang/dev/aptidus-sub2api`, customer-facing SpearRelay static app, and Sub2API backend routes.
- Goal: make SpearRelay production-connected without exposing the admin WebUI to normal users.
- Architecture decision:
  - SpearRelay is served by the same backend service at `/spearrelay/`.
  - The user-facing site calls the existing Sub2API backend on same-origin `/api/v1/customer/...`.
  - This avoids `file://` and CORS problems and keeps the admin WebUI separate.
- Code changes made:
  - `Dockerfile`: copies the `spearrelay/` static site into the runtime image at `/app/spearrelay`.
  - `backend/internal/server/routes/common.go`: serves SpearRelay from `/spearrelay/`, redirects `/spearrelay` to `/spearrelay/`, and exposes only the explicit static files used by the page. Avoid wildcard/catch-all routes here because Gin can panic on route conflicts at boot.
  - `backend/internal/web/embed_on.go`: bypasses the embedded admin frontend for `/spearrelay`, so the admin Vue app does not intercept the commercial site path.
  - `spearrelay/app.js`: removed inline click handlers and replaced them with `data-action` event delegation so the page works under the backend Content-Security-Policy.
  - `backend/internal/handler/admin/admin_service_stub_test.go`: added the missing `AdminDeleteAPIKey` test-stub method so admin handler tests compile with the existing API-key delete capability.
- Verification completed locally:
  - `node --check spearrelay/app.js` passed.
  - `rg -n "onclick=" spearrelay` returned no inline click handlers.
  - `git diff --check` passed.
  - `go test ./internal/server/routes` passed.
  - `go test ./internal/service ./internal/server ./internal/server/routes` passed.
  - `go build ./internal/handler/admin ./internal/repository ./internal/service ./internal/server ./internal/server/routes` passed.
  - `go test ./internal/service ./internal/repository ./internal/handler/admin ./internal/server ./internal/server/routes` passed.
  - `npm run typecheck` in `frontend/` passed.
  - `npm run build` in `frontend/` passed with existing Vite chunk/import warnings.
  - `go build -tags embed ./cmd/server` in `backend/` passed.
- Production deploy note:
  - First deployment of commit `84af2d17` booted badly because `/spearrelay/*filepath` conflicted with Gin route registration. The wildcard route was removed and replaced with explicit file routes before the follow-up deployment.
  - Follow-up commit `b8025eb7` deployed successfully as Railway deployment `8f9fddb3-8a9a-4bea-980d-b614ffaa354b`.
  - Live verification passed:
    - `GET https://sub2api-app-production.up.railway.app/health` returned `200 {"status":"ok"}`.
    - `GET https://sub2api-app-production.up.railway.app/spearrelay/` returned `200` and the SpearRelay HTML.
    - `GET https://sub2api-app-production.up.railway.app/spearrelay/app.js` returned `200` and the SpearRelay app JavaScript.
    - `GET https://sub2api-app-production.up.railway.app/spearrelay/styles.css` returned `200`.
    - `GET https://sub2api-app-production.up.railway.app/spearrelay/config.js` returned `200`.
    - `GET https://sub2api-app-production.up.railway.app/spearrelay` returned `301` to `/spearrelay/`.
    - `POST https://sub2api-app-production.up.railway.app/api/v1/customer/auth/login` with bad credentials returned `401 invalid email or password`, not `404`, proving the customer auth route is live.
    - Live SpearRelay HTML/JS had no `onclick=` handlers, so it is compatible with the backend CSP.
- Remaining launch toggles:
  - Production public settings currently report `registration_enabled=false` and `payment_enabled=false`.
  - That means the site is connected and live, but public signup and real customer purchases are still administratively disabled until those settings and the Stripe/payment provider setup are intentionally turned on.
- No OAuth access token, refresh token, admin key, customer API key, database password, or Stripe secret was written to this handover.

## 2026-05-06 Upstream risk controls and production safety guard

- Scope: `/Users/benzhang/dev/aptidus-sub2api` plus production Railway service `sub2api-app`.
- User clarified that the remaining Codex/Anthropic accounts are corporate accounts; only `tianyiz2020@gmail.com` should be treated as the banned personal Max/setup-token case.
- Code changes made:
  - `backend/internal/service/account.go`: `Account.IsSchedulable()` now calls `AllowsProductionTraffic()`. Anthropic `setup-token` accounts are not schedulable unless `extra.production_traffic_allowed=true`. Corporate Anthropic OAuth/API-key accounts are unaffected.
  - `backend/internal/service/ratelimit_service.go`: Anthropic upstream “out of extra usage” / usage quota style errors now set a temporary unschedulable pause instead of immediately retrying the same exhausted account.
  - `backend/internal/service/gateway_service.go`: Anthropic OAuth/setup-token selection now enforces local risk caps before scheduling. Defaults: `120` requests/5m, `2,000,000` cache-read tokens/5m, `80,000,000` total tokens/5h, `10` distinct users/5m, `10` distinct IPs/5m, and `30` minute auto-pause. Per-account overrides live in `accounts.extra` keys:
    - `risk_max_requests_5m`
    - `risk_max_cache_read_tokens_5m`
    - `risk_max_total_tokens_5h`
    - `risk_max_distinct_users_5m`
    - `risk_max_distinct_ips_5m`
    - `risk_cap_pause_minutes`
    - `risk_usage_exhausted_pause_minutes`
  - `backend/internal/repository/usage_log_repo.go` and `backend/internal/service/account_usage_service.go`: added account risk reporting from existing `usage_logs`, including account windows, cache-read tokens, internal-vs-external request counts, and top users/API keys/clients/IPs.
  - `backend/internal/handler/admin/account_handler.go` and `backend/internal/server/routes/admin.go`: added admin endpoint `GET /api/v1/admin/accounts/risk-report?platform=anthropic&hours=5&top_limit=5`.
  - `frontend/src/api/admin/accounts.ts` and `frontend/src/views/admin/AccountsView.vue`: added an admin “Risk Report” button on the account management page. It shows per-upstream account risk level, 5-minute velocity, 5-hour token concentration, internal/external request split, and top users/API keys/clients/IPs.
- Production operation completed:
  - Verified account `3|tianyiz2020@gmail.com|anthropic|setup-token|error|schedulable=false`.
  - Updated production account `3` extra flags:
    - `production_traffic_allowed=false`
    - `risk_policy=personal_setup_token_disabled`
  - Verified current Anthropic accounts after the update:
    - `id=3` remains `setup-token`, `error`, `schedulable=false`, `production_traffic_allowed=false`.
    - `id=6-10` remain Anthropic `oauth`, `active`, `schedulable=true`.
    - Existing `id=2` Anthropic OAuth account was left untouched per the user’s statement that the remaining accounts are corporate accounts.
- Verification:
  - `gofmt` passed using `/opt/homebrew/bin/gofmt`.
  - `go test ./internal/service` passed.
  - `go test ./internal/repository -run TestNonExistent` passed as a repository compile smoke.
  - `go build ./internal/handler/admin ./internal/repository ./internal/service ./internal/server` passed.
  - `npm run typecheck` in `frontend/` passed.
  - `git diff --check` passed.
  - Production read-only SQL smoke validated the two new risk-report query shapes for account windows and top user dimensions against current Anthropic account ids.
  - Full `go test ./internal/service ./internal/repository ./internal/handler/admin ./internal/server` is still blocked by a pre-existing handler test-stub mismatch from earlier dirty admin API-key work: `stubAdminService does not implement service.AdminService (missing method AdminDeleteAPIKey)`. The normal package build passes.
- No OAuth access token, refresh token, admin key, customer API key, database password, or proxy secret was written to this handover.

## 2026-05-06 Anthropic personal Max account ban investigation

- Scope: production Sub2API database via Railway service `sub2api-app`.
- User reported that upstream Anthropic account `tianyiz2020@gmail.com` was banned/refunded today, while team-organization Anthropic accounts remained normal.
- Account identity:
  - Banned account is production `accounts.id=3`, name `tianyiz2020@gmail.com`, platform `anthropic`, type `setup-token`.
  - Current status is `error`, `schedulable=false`.
  - Stored account error: `Access forbidden (403): OAuth authentication is currently not allowed for this organization.`
  - Team Anthropic accounts checked: `accounts.id=6-10`, all `platform=anthropic`, type `oauth`, status `active`, `schedulable=true`.
- Seven-day usage comparison:
  - Account `3`: `6,961` logged requests, `764,578` input tokens, `2,526,369` output tokens, `26,637,988` cache-creation tokens, `477,353,661` cache-read tokens.
  - Team accounts `6-10`: each had only `141-485` logged requests in the same query window, with `6.2M-52.8M` cache-read tokens each.
  - Account `3` had far more traffic and cache traffic than any single team account.
- Today before failure:
  - Account `3`: `976` logged successful requests from `2026-05-06 00:43:29Z` to `2026-05-06 08:31:34Z`, about `58.2M` total logged tokens.
  - Account `3` stopped successful traffic at `2026-05-06 08:31:34Z`.
  - First hard upstream failure was at `2026-05-06 08:32:19Z`: `OAuth authentication is currently not allowed for this organization.`
- Concentration:
  - Top users on account `3` in 7-day window:
    - `tianyi2020@gmail.com`, API key `13`: `3,356` requests, `132.25M` total tokens.
    - `xifengzhu520@gmail.com`, API key `9`: `2,416` requests, `312.47M` total tokens.
    - `ben.zhang.22@gmail.com`, API key `2`: `485` requests, `28.79M` total tokens.
  - Top two IPs were responsible for almost all account `3` traffic:
    - One IP: `3,859` requests, `161.36M` tokens.
    - Another IP: `2,410` requests, `312.38M` tokens.
  - Team accounts had much lower per-account concentration: `2-5` users and `3-6` IPs each in the same query window.
- Client mix:
  - Account `3` top user agents:
    - `OpenAI/Python 2.31.0`: `4,214` requests, `233.56M` tokens.
    - `claude-cli/2.1.100`: `1,286` requests, `187.54M` tokens.
    - `claude-cli/2.1.14`: `396` requests, `18.71M` tokens.
    - Other Claude CLI, Codex CLI, curl, and urllib clients also appeared.
- Model mix on account `3`:
  - `claude-haiku-4-5-20251001`: `3,882` requests.
  - `claude-sonnet-4-6`: `2,327` requests.
  - `claude-opus-4-6`: `496` requests.
  - `claude-opus-4-7`: `230` requests.
- Error pattern:
  - Account `3` had `59` `You're out of extra usage. Add more at claude.ai/settings/usage and keep going.` errors from `2026-05-05 22:58:14Z` to `2026-05-06 04:21:25Z`.
  - Team accounts also had quota-style errors, but the message was workspace/team flavored, such as `Ask your workspace admin to add more`, and they remained active.
  - The ban/revocation error was a provider-side `403`, not a normal Sub2API routing error.
- Final burst:
  - From `2026-05-06 08:20Z` to `08:31Z`, account `3` served repeated `claude-cli/2.1.92` calls for `xifengzhu520@gmail.com` / API key `9`.
  - Individual final successful calls often had `1-2` fresh input tokens and `55K-160K` cache-read tokens, meaning a cached large context was being hammered repeatedly.
  - The final minute `08:31Z` had `6` successful requests, `51,525` cache-creation tokens, and `390,801` cache-read tokens; then the provider returned the `403` organization OAuth error at `08:32:19Z`.
- Evidence-based interpretation:
  - The most likely cause is not one visible content-policy request from the logs. Successful request bodies are not stored, so content cannot be reconstructed.
  - The traffic pattern looks like commercial/proxy automation on a personal Max/setup-token account: very high request count, very high cache-read volume, multiple users/API keys/IPs, OpenAI-compatible clients plus Claude CLI clients, and repeated use after quota-style errors.
  - The team org accounts had similar technical transport/proxy/TLS settings but were org OAuth accounts and were not carrying the same seven-day concentration on one personal account.
- Recommended next actions:
  - Keep account `3` disabled; do not retry it automatically.
  - Do not route customer/commercial traffic through personal Max accounts.
  - Add per-upstream-account risk caps: maximum distinct users/IPs per upstream account, max requests per 5 minutes, max cache-read tokens per 5 minutes, max total tokens per 5-hour window, and auto-pause after repeated `out of extra usage` errors.
  - Prefer team organization accounts for production traffic and distribute load before an account hits quota/error thresholds.
  - Add a dashboard/report that highlights concentration by upstream account, user, API key, client, IP, and cache-read-token velocity.

## 2026-05-05 SpearRelay customer auth/backend connection

- Scope: `/Users/benzhang/dev/aptidus-sub2api`, focused on connecting standalone SpearRelay to the current Sub2API backend without opening admin/operator routes.
- Current production finding:
  - `spearrelay/config.js` points at `https://sub2api-app-production.up.railway.app/api/v1` and `https://sub2api-app-production.up.railway.app`.
  - Production public settings currently return `backend_mode_enabled=true`, `registration_enabled=false`, and `payment_enabled=false`.
  - Existing production `/api/v1/auth/login` intentionally blocks non-admin users while backend mode is enabled.
  - Production CORS currently rejects browser preflight from local preview origins such as `http://127.0.0.1:4177` and `Origin: null`, so a raw `file://` preview cannot complete browser login against production until the frontend is served from an allowed origin or a same-origin proxy.
- Implemented:
  - Added customer auth methods in `backend/internal/handler/auth_handler.go`: `CustomerLogin`, `CustomerLogin2FA`, and `CustomerRefreshToken`.
  - Added `backend/internal/server/routes/customer.go` with a narrow `/api/v1/customer/...` self-service surface for SpearRelay.
  - Wired customer routes in `backend/internal/server/router.go`.
  - Updated `spearrelay/app.js` to call `/api/v1/customer/auth/login`, `/api/v1/customer/auth/register`, `/api/v1/customer/auth/refresh`, `/api/v1/customer/user/profile`, `/api/v1/customer/keys`, `/api/v1/customer/payment/...`, and `/api/v1/customer/subscriptions/summary`.
  - Updated `spearrelay/README.md` with the customer endpoint list.
- Intended behavior after backend deploy:
  - Same Sub2API user table and passwords.
  - Admin users and normal users can both log in to SpearRelay if their account is active.
  - SpearRelay still exposes only billing/key/model/order/docs customer actions.
  - Sub2API admin/operator WebUI can remain backend-mode/admin-only.
- Still required before live customer browser QA:
  - Deploy the backend route changes.
  - Serve SpearRelay from the final customer domain through a same-origin backend proxy, or add the final customer origin to `cors.allowed_origins` on the Sub2API backend.
  - Enable payment config/Stripe provider if purchase flows should be live; current public settings report `payment_enabled=false`.
- Verification:
  - `node --check spearrelay/app.js`
  - `git diff --check`
  - `python3 -m http.server 4177 --directory spearrelay`
  - `curl -fsS http://127.0.0.1:4177/ >/tmp/spearrelay-index.html && curl -fsS http://127.0.0.1:4177/app.js >/tmp/spearrelay-app.js && node --check /tmp/spearrelay-app.js`
  - Live read-only probes against current production confirmed the backend-mode and CORS blockers listed above.
  - Current production `/api/v1/customer/auth/login` returns `404` before this backend change is deployed, which is expected.
- Not verified:
  - Go compile/tests were not run because this shell has no `go` or `gofmt` binary on `PATH`.
  - Customer login cannot be live-smoked until the backend route changes are deployed and CORS/same-origin hosting is configured.

## 2026-05-05 SpearRelay Apple-like UI pass

- Scope: `/Users/benzhang/dev/aptidus-sub2api/spearrelay`.
- User asked to use the `design-taste-frontend` skill and review `DavidHDev/react-bits` for UI/UX improvement, with a minimalist, translucent, Apple-like, interactive direction and less text.
- React Bits review:
  - React Bits is a React component library for animated text, UI, and backgrounds.
  - The repo license is `MIT + Commons Clause`; commercial use inside an app/site is allowed, but reselling, sublicensing, or redistributing the components themselves is restricted.
  - SpearRelay is currently a framework-free static app, so no React Bits component source was copied or imported.
- Implemented:
  - Reworked the standalone SpearRelay home/portal/docs/auth copy to be shorter and more customer-focused.
  - Replaced the heavier terminal/brutalist look with a translucent glass interface, neutral Apple-like palette, high-end sans font stack, soft mesh background, and asymmetric bento layout.
  - Added vanilla CSS/JS interaction equivalents inspired by React Bits patterns: magnetic buttons, cursor spotlight glass panels, reveal transitions, shimmer skeleton loading states, and soft perpetual motion in the product preview.
  - Preserved the existing customer-safe API wiring for signup/signin, key copy/rotation, model fetch, billing, orders, Stripe Payment Element, Turnstile, and docs.
- Boundary preserved:
  - No Sub2API backend code or existing admin WebUI files were changed in this UI pass.
  - No admin endpoints, upstream account data, provider secrets, payment secrets, or operator settings were exposed.
- Verification:
  - `node --check spearrelay/app.js`
  - `git diff --check`
  - `python3 -m http.server 4177 --directory spearrelay`
  - `curl -fsS http://127.0.0.1:4177/`
  - `curl -fsS http://127.0.0.1:4177/app.js`
  - `curl -fsS http://127.0.0.1:4177/styles.css`

## 2026-05-05 Xifeng admin promotion

- Scope: production Railway service `sub2api-app`.
- User requested `xifengzhu520@gmail.com` be checked and added as admin if it already exists.
- Access path:
  - Used the locally linked Railway project for `/Users/benzhang/dev/aptidus-sub2api`.
  - Connected to the running Railway app container with `railway ssh`.
  - Used `psql` inside the app container, so no database password or admin key was printed.
- Result:
  - Existing user found: `id=7`, `xifengzhu520@gmail.com`.
  - Before update: `role=user`, `internal_usage=false`, `status=active`.
  - Updated row to `role=admin`, `internal_usage=true`, `status=active`.
  - Verification query after update returned `role=admin`, `internal_usage=true`, `status=active`.
- Note:
  - If this user is already logged in, they may need to log out and log back in for the admin role to appear immediately.
- Secret handling:
  - No API keys, Railway token, database password, admin password, JWT, or customer API key was written to this handover.

## 2026-05-05 Xifeng password reset

- Scope: production Railway service `sub2api-app`.
- User requested a password reset for `xifengzhu520@gmail.com`.
- Action:
  - Generated a new bcrypt password hash locally with Apache `htpasswd -B`.
  - Converted the hash prefix to the Go-compatible bcrypt prefix.
  - Updated only `users.password_hash` and `users.updated_at` for the existing production user row.
- Result:
  - User row `id=7`, `xifengzhu520@gmail.com` remains `role=admin`, `internal_usage=true`, `status=active`.
  - Verification query showed a 60-character bcrypt hash on the user row.
  - Login smoke check against `https://sub2api-app-production.up.railway.app/api/v1/auth/login` returned HTTP `200`, `code=0`, `role=admin`, `status=active`.
- Caveat:
  - Production DB currently does not have a `token_version` column, so this direct reset could not increment token version to force-expire existing sessions. The user can log in with the new password; existing sessions may last until their normal expiry depending on the deployed auth code.
- Secret handling:
  - The temporary password was not written to this handover.
  - No JWT, refresh token, admin key, database password, Railway token, or customer API key was written to this handover.

## 2026-05-05 SpearRelay standalone customer site

- Scope: `/Users/benzhang/dev/aptidus-sub2api/spearrelay`.
- User clarified SpearRelay must be separate from the Sub2API WebUI. Sub2API should remain the admin/operator system for users, upstream accounts, models, routing, and payment-provider settings. SpearRelay should be the premium customer-facing commercial site, with Sub2API only acting as the invisible backend.
- Correction:
  - Removed the earlier in-WebUI `/spearrelay` route/page approach.
  - Restored the existing Sub2API frontend route/test/register changes from that approach.
- Implemented:
  - Added standalone static app under `spearrelay/`.
  - Added `spearrelay/index.html`, `spearrelay/styles.css`, `spearrelay/app.js`, `spearrelay/config.js`, `spearrelay/config.example.js`, and `spearrelay/README.md`.
  - SpearRelay has its own premium visual system and does not import or reuse the Sub2API Vue admin WebUI components, router, layout, Tailwind theme, or i18n.
  - SpearRelay calls only existing customer-safe Sub2API endpoints for auth, profile, keys, key rotation, checkout, orders, subscriptions, and model discovery.
  - SpearRelay reads public settings and supports Turnstile on signin/signup if Sub2API has Turnstile enabled.
  - No new Sub2API backend endpoint was added in this pass.
- Customer capabilities in the standalone app:
  - Signup/signin.
  - Customer portal with balance, provisioned API key, key copy, key rotation, available-model fetch, token top-up, subscription plan purchase, recent orders, and API docs.
  - Stripe top-up PaymentIntent support through Stripe Payment Element when the backend returns a `client_secret`.
  - Stripe subscription checkout redirect when the backend returns a `pay_url`.
- Boundary preserved:
  - No admin endpoints are used.
  - No upstream account data, provider secrets, admin keys, cost internals, group management, model routing controls, or Sub2API operator settings are exposed.
  - `config.js` contains only public browser config; it must never contain Sub2API admin keys, Stripe secret keys, OAuth tokens, or upstream credentials.
- Verification:
  - `node --check spearrelay/app.js`
  - `git diff -- frontend/src/router/index.ts frontend/src/router/__tests__/guards.spec.ts frontend/src/router/README.md frontend/src/views/auth/RegisterView.vue frontend/src/views/SpearRelayLandingView.vue` returned no diff, confirming the earlier in-WebUI route changes were removed.
  - `git diff --check`
  - `python3 -m http.server 4177 --directory spearrelay`
  - `curl -fsS http://127.0.0.1:4177/`
  - `curl -fsS http://127.0.0.1:4177/app.js >/tmp/spearrelay-app.js && node --check /tmp/spearrelay-app.js`

## 2026-05-03 SpearAgent local install 503 diagnosis

- Scope: local SpearAgent install at `/Users/benzhang/SpearAgent` plus production Sub2API.
- User reported a fresh Mac Studio SpearAgent error:
  - `503 {"error":{"type":"service_unavailable","message":"Proxy is currently disabled. Enable it from the dashboard."}}`
- Diagnosis:
  - This message comes from the old Spear Proxy production endpoint, not from the newly created `beib70812@gmail.com` Sub2API key.
  - The old Spear Proxy production service was intentionally disabled after upstream accounts were migrated to Sub2API.
  - Local `/Users/benzhang/.spearagent-webui/local-provider.env` has Sub2API configured and the WebUI route API is live at `localhost:25809`.
  - `GET /api/managed-model-routing/options?backend=claude` returns `sub2apiAvailable=true` and Sub2API models including `claude-opus-4-7`, `claude-sonnet-4-6`, and `claude-haiku-4-5-20251001`.
  - Recent local conversation logs still showed `currentModelId: "default"` and legacy Claude/SpearProxy model entries, so the failing conversation was using the legacy default route rather than an explicit Sub2API managed route.
- User-facing guidance:
  - Start a new conversation and choose an explicit Sub2API route from the model/route selector, for example `claude-sonnet-4-6` or `claude-opus-4-7`.
  - Avoid the plain `Default`/legacy Claude route until SpearAgent is patched to remove the disabled SpearProxy fallback.
- No API keys, admin credentials, or OAuth tokens were written to this handover.

## 2026-05-03 Test user provisioning: beib70812@gmail.com

- Scope: production Sub2API at `https://sub2api-app-production.up.railway.app`.
- User requested a test user for `beib70812@gmail.com` in the `default` group and asked for the API key.
- Result:
  - The user already existed, so it was reused rather than duplicated.
  - Production user id: `8`.
  - Default group id: `1`.
  - The user was updated as a normal non-admin user with `internal_usage=true`.
  - A fresh API key was created for this user.
  - API key id: `12`.
  - API key `internal_usage=true`.
- Verification:
  - Authenticated `/v1/models` with the new API key returned HTTP `200`.
  - The key saw `6` available models.
- Secret handling:
  - The newly issued API key was returned to the user in-chat per request.
  - The full API key, admin password, admin JWT, and Railway secrets were not written to this handover.

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
