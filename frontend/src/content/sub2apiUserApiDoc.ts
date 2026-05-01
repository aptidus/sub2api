export const sub2apiUserApiDoc = `# Sub2API User API Guide

This is the customer-facing API guide. It intentionally does not include admin keys, upstream account details, provider secrets, or internal cost data.

## Quick Start

1. Sign up or sign in.
2. Open **API Keys** to copy the key provisioned for your account.
3. Rotate the key from **API Keys** if it is exposed or you need to replace it.
4. Call the model endpoint that matches your client.

\`\`\`http
Authorization: Bearer $SUB2API_KEY
\`\`\`

Recommended environment variables:

\`\`\`bash
export SUB2API_BASE_URL="https://sub2api-app-production.up.railway.app"
export SUB2API_KEY="sk-your-user-api-key"
\`\`\`

Do not put API keys in URLs, browser local storage, public GitHub repos, or client-side code. If a key is exposed, rotate it immediately from **API Keys**.

## Base URLs

Use the origin by default:

\`\`\`text
https://sub2api-app-production.up.railway.app
\`\`\`

China/APAC relay, if you explicitly need it:

\`\`\`text
https://sub2api-cn-relay-production.up.railway.app
\`\`\`

## List Models

\`\`\`bash
curl -sS "$SUB2API_BASE_URL/v1/models" \\
  -H "Authorization: Bearer $SUB2API_KEY"
\`\`\`

The response contains model IDs. Use the \`id\` value in later requests.

\`\`\`json
{
  "object": "list",
  "data": [
    {
      "id": "claude-sonnet-4-6",
      "object": "model",
      "created": 0,
      "owned_by": "sub2api"
    }
  ]
}
\`\`\`

## Anthropic Messages API

Use this for Anthropic-compatible clients such as Claude Code, OpenCode, or direct Messages API integrations.

\`\`\`http
POST /v1/messages
Authorization: Bearer $SUB2API_KEY
Content-Type: application/json
anthropic-version: 2023-06-01
\`\`\`

Minimal request:

\`\`\`bash
curl -sS "$SUB2API_BASE_URL/v1/messages" \\
  -H "Authorization: Bearer $SUB2API_KEY" \\
  -H "Content-Type: application/json" \\
  -H "anthropic-version: 2023-06-01" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 800,
    "messages": [
      {
        "role": "user",
        "content": "Reply with a short answer."
      }
    ]
  }'
\`\`\`

Streaming:

\`\`\`json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 800,
  "stream": true,
  "messages": [
    {
      "role": "user",
      "content": "Write a short project plan."
    }
  ]
}
\`\`\`

Reasoning effort for Messages:

\`\`\`json
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
\`\`\`

Allowed Messages effort values:

\`\`\`text
low
medium
high
max
\`\`\`

## OpenAI Chat Completions API

Use this for OpenAI-compatible SDKs and tools that call Chat Completions.

\`\`\`http
POST /v1/chat/completions
Authorization: Bearer $SUB2API_KEY
Content-Type: application/json
\`\`\`

Minimal request:

\`\`\`bash
curl -sS "$SUB2API_BASE_URL/v1/chat/completions" \\
  -H "Authorization: Bearer $SUB2API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "messages": [
      {
        "role": "user",
        "content": "Reply with a short answer."
      }
    ]
  }'
\`\`\`

Reasoning effort for Chat Completions:

\`\`\`json
{
  "model": "claude-sonnet-4-6",
  "reasoning_effort": "high",
  "messages": [
    {
      "role": "user",
      "content": "Compare these two options and recommend one."
    }
  ]
}
\`\`\`

Allowed Chat Completions effort values:

\`\`\`text
low
medium
high
xhigh
\`\`\`

## OpenAI Responses API

Use this for clients built around OpenAI's newer Responses format.

\`\`\`http
POST /v1/responses
Authorization: Bearer $SUB2API_KEY
Content-Type: application/json
\`\`\`

Minimal request:

\`\`\`bash
curl -sS "$SUB2API_BASE_URL/v1/responses" \\
  -H "Authorization: Bearer $SUB2API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "input": "Reply with a short answer."
  }'
\`\`\`

Reasoning effort for Responses:

\`\`\`json
{
  "model": "claude-sonnet-4-6",
  "reasoning": {
    "effort": "high"
  },
  "input": "Analyze the tradeoffs."
}
\`\`\`

Allowed Responses effort values:

\`\`\`text
low
medium
high
xhigh
\`\`\`

## Count Tokens

Use this to estimate request size before sending a larger Messages request.

\`\`\`bash
curl -sS "$SUB2API_BASE_URL/v1/messages/count_tokens" \\
  -H "Authorization: Bearer $SUB2API_KEY" \\
  -H "Content-Type: application/json" \\
  -H "anthropic-version: 2023-06-01" \\
  -d '{
    "model": "claude-sonnet-4-6",
    "messages": [
      {
        "role": "user",
        "content": "Count these tokens."
      }
    ]
  }'
\`\`\`

## Usage

Use this to inspect usage for the authenticated user API key.

\`\`\`bash
curl -sS "$SUB2API_BASE_URL/v1/usage" \\
  -H "Authorization: Bearer $SUB2API_KEY"
\`\`\`

## Tool Calling Notes

Tool calling is supported on compatible endpoints.

For Chat Completions, avoid forcing a specific tool while also enabling high reasoning. If a request forces tool use and enables reasoning, upstream may reject it with a message like:

\`\`\`text
Thinking may not be enabled when tool_choice forces tool use.
\`\`\`

Use \`tool_choice: "auto"\` or remove reasoning for forced-tool calls.

## Security Rules

- Treat \`sk-...\` API keys like passwords.
- Rotate a key immediately if it is exposed.
- Create separate keys for separate projects.
- Delete unused keys.
- Do not share admin keys with users or client applications.
- User API calls only need the user's own API key.
`
