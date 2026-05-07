# SpearRelay

SpearRelay is the standalone customer-facing commercial site for Sub2API-backed API access.

It is intentionally separate from the existing Sub2API WebUI:

- Sub2API WebUI: admin/operator management for users, upstream accounts, models, groups, pricing, payment providers, and internal reporting.
- SpearRelay: customer signup, signin, token purchases, subscriptions, provisioned-key rotation, available-model visibility, order history, and API docs.

## Files

- `index.html` - standalone static entrypoint.
- `styles.css` - SpearRelay-only premium visual system.
- `app.js` - framework-free customer portal and Sub2API API client.
- `config.js` - deployment config used by the browser.
- `config.example.js` - safe template for environment-specific config.

## UI Direction

The current design is intentionally independent from the Sub2API admin WebUI. It uses a minimalist translucent style with glass panels, soft motion, magnetic buttons, spotlight hover panels, skeleton loading states, and an asymmetric bento layout.

React Bits was reviewed as inspiration for interaction patterns, but no React Bits component source is copied or imported here. SpearRelay is currently a framework-free static app, and React Bits is a React component library with a Commons Clause restriction against reselling or redistributing the components themselves.

## Local Preview

From this repository root:

```bash
python3 -m http.server 4177 --directory spearrelay
```

Then open:

```text
http://127.0.0.1:4177
```

## Backend Configuration

`config.js` controls the invisible Sub2API backend connection:

```js
window.SPEARRELAY_CONFIG = {
  apiBaseUrl: 'https://sub2api-app-production.up.railway.app/api/v1',
  gatewayBaseUrl: 'https://sub2api-app-production.up.railway.app',
  supportEmail: 'support@spearrelay.com'
}
```

Do not put Sub2API admin keys, upstream OAuth tokens, Stripe secret keys, or provider credentials in this file. It is public browser code.

## Customer-Safe Calls

SpearRelay uses customer-safe Sub2API endpoints. These endpoints share the same user table as the admin WebUI but expose only customer self-service actions:

- `GET /api/v1/settings/public`
- `POST /api/v1/customer/auth/login`
- `POST /api/v1/customer/auth/register`
- `POST /api/v1/customer/auth/refresh`
- `GET /api/v1/customer/user/profile`
- `GET /api/v1/customer/keys`
- `POST /api/v1/customer/keys/:id/rotate`
- `GET /api/v1/customer/payment/checkout-info`
- `POST /api/v1/customer/payment/orders`
- `GET /api/v1/customer/payment/orders/my`
- `GET /v1/models` with the customer API key

No admin endpoints are used.
