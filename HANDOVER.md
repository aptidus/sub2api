# Sub2API Handover

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
