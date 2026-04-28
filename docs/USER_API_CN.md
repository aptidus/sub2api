# Sub2API 用户 API 文档

本文只面向模型接口使用者。用户只需要知道怎么鉴权、怎么调用接口、怎么选择模型和推理强度。

## 1. 基础信息

生产地址：

```text
https://sub2api-app-production.up.railway.app
```

鉴权方式：

```http
Authorization: Bearer $SUB2API_KEY
```

推荐把 key 放在环境变量里：

```bash
export SUB2API_BASE_URL="https://sub2api-app-production.up.railway.app"
export SUB2API_KEY="你的用户 API Key"
```

不要把 key 放进 URL 查询参数里。

## 2. 可用模型

以接口实时返回为准：

```bash
curl -sS "$SUB2API_BASE_URL/v1/models" \
  -H "Authorization: Bearer $SUB2API_KEY"
```

返回示例：

```json
{
  "object": "list",
  "data": [
    {
      "id": "claude-opus-4-7",
      "object": "model",
      "created": 0,
      "owned_by": "sub2api"
    },
    {
      "id": "claude-sonnet-4-6",
      "object": "model",
      "created": 0,
      "owned_by": "sub2api"
    }
  ]
}
```

调用时只需要使用 `id` 字段。系统会在后台自动处理排队、重试和可用账号切换。

## 3. Claude Messages 接口

适合 Claude Code、OpenCode、Claude SDK 或兼容 Anthropic Messages 格式的客户端。

Endpoint：

```text
POST /v1/messages
```

最小请求：

```bash
curl -sS "$SUB2API_BASE_URL/v1/messages" \
  -H "Authorization: Bearer $SUB2API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 800,
    "messages": [
      {
        "role": "user",
        "content": "用三句话解释什么是现金流。"
      }
    ]
  }'
```

开启流式输出：

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 800,
  "stream": true,
  "messages": [
    {
      "role": "user",
      "content": "写一个简短的项目计划。"
    }
  ]
}
```

### Claude 格式的 reasoning

推荐使用 `output_config.effort`：

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
      "content": "分析这个方案的主要风险，并给出优先级。"
    }
  ]
}
```

可用值：

```text
low
medium
high
max
```

`max` 会被视为最高推理强度。简单任务建议 `low` 或 `medium`；复杂规划、代码、长文分析建议 `high` 或 `max`。

## 4. OpenAI Chat Completions 接口

适合 OpenAI SDK、LangChain、LlamaIndex、Dify、Cherry Studio 等使用 Chat Completions 格式的客户端。

Endpoint：

```text
POST /v1/chat/completions
```

最小请求：

```bash
curl -sS "$SUB2API_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $SUB2API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-6",
    "messages": [
      {
        "role": "user",
        "content": "写一封中文商务邮件，语气友好但直接。"
      }
    ]
  }'
```

### Chat Completions reasoning

使用 `reasoning_effort`：

```json
{
  "model": "claude-sonnet-4-6",
  "reasoning_effort": "high",
  "messages": [
    {
      "role": "user",
      "content": "比较两个增长方案，并推荐一个。"
    }
  ]
}
```

可用值：

```text
low
medium
high
xhigh
```

## 5. OpenAI Responses 接口

适合使用 OpenAI Responses 格式的新版客户端。

Endpoint：

```text
POST /v1/responses
```

最小请求：

```bash
curl -sS "$SUB2API_BASE_URL/v1/responses" \
  -H "Authorization: Bearer $SUB2API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-6",
    "input": "把这句话改得更专业：我们明天再聊。"
  }'
```

### Responses reasoning

使用 `reasoning.effort`：

```json
{
  "model": "claude-sonnet-4-6",
  "reasoning": {
    "effort": "high",
    "summary": "auto"
  },
  "input": "制定一个三步执行方案。"
}
```

可用值：

```text
low
medium
high
xhigh
```

如果客户端会调用 compact，也支持：

```text
POST /v1/responses/compact
```

## 6. Tool Call / Agentic 用法

三个主接口都支持工具调用。基本流程是：

1. 客户端把工具定义发给模型。
2. 模型返回要调用的工具名和参数。
3. 客户端执行工具。
4. 客户端把工具结果发回模型，让模型继续回答。

### Claude Messages 工具示例

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 1000,
  "tools": [
    {
      "name": "get_weather",
      "description": "查询城市天气",
      "input_schema": {
        "type": "object",
        "properties": {
          "city": {
            "type": "string"
          }
        },
        "required": ["city"]
      }
    }
  ],
  "messages": [
    {
      "role": "user",
      "content": "查一下上海今天的天气，并给我一句出门建议。"
    }
  ]
}
```

工具结果回传示例：

```json
{
  "model": "claude-sonnet-4-6",
  "max_tokens": 1000,
  "messages": [
    {
      "role": "user",
      "content": "查一下上海今天的天气，并给我一句出门建议。"
    },
    {
      "role": "assistant",
      "content": [
        {
          "type": "tool_use",
          "id": "toolu_01",
          "name": "get_weather",
          "input": {
            "city": "上海"
          }
        }
      ]
    },
    {
      "role": "user",
      "content": [
        {
          "type": "tool_result",
          "tool_use_id": "toolu_01",
          "content": "上海今天 18-24 度，小雨。"
        }
      ]
    }
  ]
}
```

### Chat Completions 工具示例

```json
{
  "model": "claude-sonnet-4-6",
  "messages": [
    {
      "role": "user",
      "content": "查一下上海今天的天气。"
    }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "查询城市天气",
        "parameters": {
          "type": "object",
          "properties": {
            "city": {
              "type": "string"
            }
          },
          "required": ["city"]
        }
      }
    }
  ]
}
```

工具结果回传示例：

```json
{
  "model": "claude-sonnet-4-6",
  "messages": [
    {
      "role": "user",
      "content": "查一下上海今天的天气。"
    },
    {
      "role": "assistant",
      "tool_calls": [
        {
          "id": "call_01",
          "type": "function",
          "function": {
            "name": "get_weather",
            "arguments": "{\"city\":\"上海\"}"
          }
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_01",
      "content": "上海今天 18-24 度，小雨。"
    }
  ]
}
```

### Responses 工具示例

```json
{
  "model": "claude-sonnet-4-6",
  "input": "查一下上海今天的天气。",
  "tools": [
    {
      "type": "function",
      "name": "get_weather",
      "description": "查询城市天气",
      "parameters": {
        "type": "object",
        "properties": {
          "city": {
            "type": "string"
          }
        },
        "required": ["city"]
      }
    }
  ]
}
```

## 7. Token 统计

Claude Messages 格式可用：

```text
POST /v1/messages/count_tokens
```

示例：

```bash
curl -sS "$SUB2API_BASE_URL/v1/messages/count_tokens" \
  -H "Authorization: Bearer $SUB2API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-6",
    "messages": [
      {
        "role": "user",
        "content": "统计这句话大概用了多少 token。"
      }
    ]
  }'
```

## 8. 用量查询

```bash
curl -sS "$SUB2API_BASE_URL/v1/usage" \
  -H "Authorization: Bearer $SUB2API_KEY"
```

## 9. 常见错误

`401 authentication_error`：API Key 错误、过期、禁用，或没有放在 `Authorization: Bearer` 里。

`403 permission_error`：当前 key 没有权限访问该接口。

`404 not_found`：接口路径写错。

`429 rate_limit_error`：请求过多或并发过高。稍后重试即可。

`400 invalid_request_error`：请求 JSON 格式、字段类型、上下文长度或工具消息顺序不符合要求。

`502/503 api_error`：当前没有可用模型账号，或所有可用账号都暂时失败。系统会先自动重试和切换；如果最后仍失败，才会返回错误。

## 10. 客户端配置速查

Claude Code / OpenCode：

```bash
export ANTHROPIC_BASE_URL="https://sub2api-app-production.up.railway.app"
export ANTHROPIC_AUTH_TOKEN="$SUB2API_KEY"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
```

OpenAI SDK：

```python
from openai import OpenAI

client = OpenAI(
    api_key="你的用户 API Key",
    base_url="https://sub2api-app-production.up.railway.app/v1",
)

response = client.chat.completions.create(
    model="claude-sonnet-4-6",
    messages=[
        {"role": "user", "content": "用中文写一句产品介绍。"}
    ],
)

print(response.choices[0].message.content)
```

Node.js OpenAI SDK：

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.SUB2API_KEY,
  baseURL: "https://sub2api-app-production.up.railway.app/v1",
});

const response = await client.chat.completions.create({
  model: "claude-sonnet-4-6",
  messages: [
    { role: "user", content: "用中文写一句产品介绍。" },
  ],
});

console.log(response.choices[0].message.content);
```
