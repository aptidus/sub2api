# SpearAgent Sub2API Integration Handoff

Date: 2026-04-28

This handoff is for the Codex agent working in the SpearAgent project.

Goal: SpearAgent should remain a standalone product with two backend choices:

- Default: SpearProxy. This stays the default managed backend and should not expose upstream model details.
- Optional model selection: Sub2API. This lets a SpearAgent user pick a Sub2API model and reasoning effort.

SpearProxy and Sub2API are separate services. Both serve SpearAgent, but neither should be treated as exclusive to SpearAgent.

## Live Sub2API Base URLs

Origin:

```text
https://sub2api-app-production.up.railway.app
```

China/APAC front door relay:

```text
https://sub2api-cn-relay-production.up.railway.app
```

Use the origin by default. Use the relay for users in China/APAC if SpearAgent has a region toggle or latency-based routing.

Do not expose the Sub2API admin key to any client. Only SpearAgent server-side code may call Sub2API admin APIs.

## Public User API Auth

Sub2API model calls accept:

```http
Authorization: Bearer <sub2api-user-api-key>
```

The current user keys are `sk-...` keys. SpearAgent should store the key encrypted server-side and only inject it from backend/agent runtime code. Do not put it in browser local storage if avoidable.

## Public Model Endpoints

### List models

```http
GET /v1/models
Authorization: Bearer <sub2api-user-api-key>
```

Current live response returns these model IDs:

```text
claude-opus-4-5-20251101
claude-opus-4-6
claude-opus-4-7
claude-sonnet-4-6
claude-sonnet-4-5-20250929
claude-haiku-4-5-20251001
```

Recommended SpearAgent model selector behavior:

- Show Sub2API models only when backend mode is `sub2api`.
- Keep SpearProxy mode as `Default` with no direct upstream model selector.
- Prefer `claude-sonnet-4-6` as the default Sub2API model.
- Offer `claude-opus-4-7` for complex/high-reasoning tasks.
- Keep `claude-haiku-4-5-20251001` hidden or behind an experimental flag for now because live QA saw an upstream 502 on this model.
- Treat older model IDs as selectable only if Ben explicitly wants them exposed; they are live, but they may confuse users.

### Anthropic Messages

```http
POST /v1/messages
Authorization: Bearer <sub2api-user-api-key>
Content-Type: application/json
anthropic-version: 2023-06-01
```

Minimal request:

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 800,
  "messages": [
    {
      "role": "user",
      "content": "Reply with a short answer."
    }
  ]
}
```

Reasoning effort for Messages:

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 1200,
  "output_config": {
    "effort": "high"
  },
  "messages": [
    {
      "role": "user",
      "content": "Analyze the tradeoffs."
    }
  ]
}
```

Allowed effort values:

```text
low
medium
high
max
```

Tool call request:

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 1000,
  "tools": [
    {
      "name": "get_weather",
      "description": "Get weather for a city",
      "input_schema": {
        "type": "object",
        "properties": {
          "city": { "type": "string" }
        },
        "required": ["city"]
      }
    }
  ],
  "tool_choice": {
    "type": "tool",
    "name": "get_weather"
  },
  "messages": [
    {
      "role": "user",
      "content": "Use the tool for San Francisco."
    }
  ]
}
```

### OpenAI Chat Completions

```http
POST /v1/chat/completions
Authorization: Bearer <sub2api-user-api-key>
Content-Type: application/json
```

Minimal request:

```json
{
  "model": "claude-sonnet-4-6",
  "messages": [
    {
      "role": "user",
      "content": "Reply with a short answer."
    }
  ]
}
```

Reasoning effort for Chat Completions:

```json
{
  "model": "claude-sonnet-4-6",
  "reasoning_effort": "high",
  "messages": [
    {
      "role": "user",
      "content": "Analyze the tradeoffs."
    }
  ]
}
```

Allowed effort values:

```text
low
medium
high
xhigh
```

Tool call request:

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 1000,
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get weather for a city",
        "parameters": {
          "type": "object",
          "properties": {
            "city": { "type": "string" }
          },
          "required": ["city"]
        }
      }
    }
  ],
  "tool_choice": "auto",
  "messages": [
    {
      "role": "user",
      "content": "Use the tool for Tokyo."
    }
  ]
}
```

Important QA finding: `reasoning_effort` plus forced tool choice can be rejected if the request forces immediate tool use. In live QA, `tool_choice: "auto"` plus `reasoning_effort: "high"` worked, and forced tool use without reasoning worked.

### OpenAI Responses

```http
POST /v1/responses
Authorization: Bearer <sub2api-user-api-key>
Content-Type: application/json
```

Minimal request:

```json
{
  "model": "claude-sonnet-4-6",
  "input": "Reply with a short answer."
}
```

Reasoning effort for Responses:

```json
{
  "model": "claude-sonnet-4-6",
  "input": "Analyze the tradeoffs.",
  "reasoning": {
    "effort": "high",
    "summary": "auto"
  }
}
```

Allowed effort values:

```text
low
medium
high
xhigh
```

### Token count and usage

Anthropic token count:

```http
POST /v1/messages/count_tokens
Authorization: Bearer <sub2api-user-api-key>
Content-Type: application/json
```

Usage:

```http
GET /v1/usage
Authorization: Bearer <sub2api-user-api-key>
```

## Recommended Reasoning Defaults

Use this in SpearAgent UI:

| UI label | Messages value | Chat/Responses value |
|---|---|---|
| Low | `low` | `low` |
| Medium | `medium` | `medium` |
| High | `high` | `high` |
| Max / Extra High | `max` | `xhigh` |

Model defaults:

| Model | Suggested default effort | Notes |
|---|---:|---|
| `claude-sonnet-4-6` | `medium` | Recommended default Sub2API model. |
| `claude-opus-4-7` | `high` | Best current complex-task model. |
| `claude-sonnet-4-5-20250929` | `medium` | Works, but older. Hide unless needed. |
| `claude-opus-4-6` | `high` | Works, but older than Opus 4.7. Hide unless needed. |
| `claude-opus-4-5-20251101` | `high` | Works, but older. Hide unless needed. |
| `claude-haiku-4-5-20251001` | `low` | Live QA saw upstream 502 once. Do not make default. |

## SpearAgent Product Behavior

Add a backend/provider selector:

```text
Default (SpearProxy)
Sub2API Models
```

Default mode:

- Uses SpearProxy.
- No model selector.
- No upstream model info exposed.

Sub2API mode:

- Uses Sub2API.
- Shows model selector.
- Shows reasoning effort selector.
- Calls `/v1/models` periodically or on settings open to refresh the selectable model list.
- Should preserve user-selected model and effort per workspace/conversation if SpearAgent already has settings persistence.

## Sub2API Auto-Provisioning Contract

SpearAgent should auto-provision a Sub2API user and Sub2API key when a SpearAgent user signs up or first enables Sub2API mode.

Server-side env vars to add to SpearAgent:

```bash
SUB2API_BASE_URL=https://sub2api-app-production.up.railway.app
SUB2API_CN_RELAY_BASE_URL=https://sub2api-cn-relay-production.up.railway.app
SUB2API_ADMIN_KEY=<server-only admin key>
SUB2API_DEFAULT_GROUP_ID=1
SUB2API_DEFAULT_BALANCE=1000
SUB2API_DEFAULT_KEY_QUOTA=1000
SUB2API_DEFAULT_CONCURRENCY=4
SUB2API_DEFAULT_RPM_LIMIT=0
```

Suggested SpearAgent database fields:

```text
users.sub2api_user_id
users.sub2api_api_key_id
users.sub2api_api_key_encrypted
users.sub2api_default_model
users.sub2api_default_reasoning_effort
users.sub2api_region_mode         // origin | cn_relay | auto
users.sub2api_provisioned_at
```

Provisioning algorithm:

1. On signup or first Sub2API use, check whether the SpearAgent user already has `sub2api_user_id` and an encrypted Sub2API key.
2. If not, search Sub2API admin users by email to avoid duplicate users.
3. If no matching Sub2API user exists, create one.
4. Create a user-owned Sub2API API key under group `1`.
5. Store the Sub2API user ID, API key ID, and encrypted API key in SpearAgent.
6. Never send the Sub2API admin key to the frontend.
7. Prefer idempotency guards so repeated signup retries do not create duplicate Sub2API users.

### Create Sub2API user

```http
POST /api/v1/admin/users
x-api-key: <SUB2API_ADMIN_KEY>
Content-Type: application/json
```

Body:

```json
{
  "email": "user@example.com",
  "password": "generated-random-password-not-shown-to-user",
  "username": "Display Name",
  "notes": "created by SpearAgent auto-provisioning",
  "balance": 1000,
  "concurrency": 4,
  "rpm_limit": 0,
  "allowed_groups": [1]
}
```

Sub2API creates this as a normal user, not an admin.

### Create user-owned API key

```http
POST /api/v1/admin/users/{sub2api_user_id}/api-keys
x-api-key: <SUB2API_ADMIN_KEY>
Content-Type: application/json
```

Body:

```json
{
  "name": "SpearAgent",
  "group_id": 1,
  "quota": 1000
}
```

The response contains the generated `sk-...` key. Store it encrypted. Do not show it in normal UI unless Ben explicitly wants an advanced/debug panel.

### Get a user's Sub2API keys

```http
GET /api/v1/admin/users/{sub2api_user_id}/api-keys
x-api-key: <SUB2API_ADMIN_KEY>
```

### Update balance later

```http
POST /api/v1/admin/users/{sub2api_user_id}/balance
x-api-key: <SUB2API_ADMIN_KEY>
Content-Type: application/json
```

Body:

```json
{
  "balance": 1000,
  "operation": "set",
  "notes": "SpearAgent testing allocation"
}
```

For current testing, default balance and key quota are both `1000`. For public production, lower this or connect it to billing/subscription limits.

## Live QA Results From 2026-04-28

Environment:

- Origin: `https://sub2api-app-production.up.railway.app`
- Relay: `https://sub2api-cn-relay-production.up.railway.app`
- Active Anthropic account: account `3`, setup-token, Oxylabs proxy attached, TLS fingerprinting enabled.
- No temporary QA users remain after cleanup.

Endpoint QA:

| Test | Result | Latency |
|---|---:|---:|
| Origin `GET /v1/models` | 200 | 0.34s |
| Relay `GET /v1/models` | 200 | 0.91s |
| Messages + reasoning high, Sonnet 4.6 | 200 | 1.57s |
| Messages + forced tool call, Sonnet 4.6 | 200 | 2.04s |
| Chat Completions + forced tool call, no reasoning | 200 | 1.63s |
| Chat Completions + reasoning, no tool | 200 | 1.73s |
| Chat Completions + reasoning + `tool_choice: auto` | 200 | 2.02s |
| Chat Completions + reasoning + forced tool choice | 400 | 0.47s |
| Responses + reasoning high, Sonnet 4.6 | 200 | 2.64s |
| Relay Messages, Sonnet 4.6 | 200 | 2.39s |
| Auto-provision temp user + key + `/v1/models` | 200 | cleaned up |

Model smoke:

| Model | Result | Latency | Notes |
|---|---:|---:|---|
| `claude-sonnet-4-6` | 200 | 2.83s | Works. |
| `claude-opus-4-7` | 200 | 1.84s | Works. |
| `claude-opus-4-6` | 200 | 2.17s | Works. |
| `claude-opus-4-5-20251101` | 200 | 1.78s | Works. |
| `claude-sonnet-4-5-20250929` | 200 | 1.94s | Works. |
| `claude-haiku-4-5-20251001` | 502 | 1.23s | Upstream unavailable in this run. Hide or mark experimental. |

Observed transient issue:

- A first feature QA pass saw several upstream `500` / `502` errors from Anthropic through the single available account.
- Retrying shortly after succeeded for Sonnet 4.6, Opus 4.7, and most other models.
- This is not an auth failure and not a Sub2API key failure. Logs showed upstream Anthropic `500 Internal server error`, and Sub2API had no second Anthropic account to fail over to.
- For SpearAgent UX, show a retryable provider error when Sub2API returns `502` or `500` with upstream/service exhausted wording. Do not tell users their API key is invalid unless the status is `401` or explicit auth failure.

## Implementation Notes For SpearAgent

- Use SpearProxy as default backend.
- Add Sub2API as an explicit selectable backend.
- Do not merge SpearProxy and Sub2API model lists.
- Do not expose Sub2API admin key client-side.
- Do not expose upstream account details to users.
- Sub2API model selector should use model IDs from `GET /v1/models`.
- For China users, route model calls through `SUB2API_CN_RELAY_BASE_URL`; admin provisioning should still call the origin unless there is a strong reason to relay admin traffic.
- If SpearAgent supports multiple agent engines, map them like this:
  - Claude/Anthropic-native agents: `/v1/messages`
  - OpenAI SDK / generic chat engines: `/v1/chat/completions`
  - OpenAI Responses/Codex-style engines: `/v1/responses`
- Sub2API converts protocol formats but does not execute tools. SpearAgent remains responsible for executing tool calls and sending tool results back.

