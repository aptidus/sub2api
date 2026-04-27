# Sub2API China/APAC Front-Door Relay

This is a stateless reverse proxy for users who need a closer front door to
Sub2API from mainland China or nearby APAC networks.

It does not change upstream provider routing. Anthropic/OpenAI/Gemini upstream
accounts still use their own Sub2API account proxy settings.

## Runtime

- Base image: `caddy:2-alpine`
- Health check: `GET /relay-health`
- Origin: `SUB2API_ORIGIN`, defaulting to
  `https://sub2api-app-production.up.railway.app`

## Why This Is Separate From Account Proxies

Sub2API account proxies control the server-to-provider leg, for example
Sub2API to Anthropic. This relay controls only the user-to-Sub2API leg.

Keeping those two paths separate avoids accidentally changing the IP seen by
upstream OAuth providers while still giving China users a closer API base URL.
