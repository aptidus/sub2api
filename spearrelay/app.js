const config = {
  apiBaseUrl: '/api/v1',
  gatewayBaseUrl: '',
  supportEmail: 'support@spearrelay.com',
  ...(window.SPEARRELAY_CONFIG || {})
}

const storage = {
  access: 'spearrelay_access_token',
  refresh: 'spearrelay_refresh_token',
  user: 'spearrelay_user',
  apiBase: 'spearrelay_api_base_url'
}

const state = {
  view: location.hash.replace('#', '') || 'home',
  authMode: 'signin',
  loading: false,
  message: '',
  error: '',
  user: readJson(storage.user),
  settings: null,
  profile: null,
  keys: [],
  models: [],
  checkout: null,
  orders: [],
  subscriptions: null,
  paymentIntent: null,
  stripeMounted: false,
  verifyCodeSent: false,
  turnstileToken: '',
  turnstileWidgetId: null
}

const app = document.querySelector('#app')
window.state = state
window.render = render

window.addEventListener('hashchange', () => {
  state.view = location.hash.replace('#', '') || 'home'
  state.message = ''
  state.error = ''
  render()
  if (state.view === 'portal' && isAuthed()) {
    loadPortal()
  }
})

document.addEventListener('click', async (event) => {
  const target = event.target.closest('[data-action]')
  if (!target) return

  const action = target.dataset.action
  if (action === 'nav') {
    location.hash = target.dataset.view || 'home'
  }
  if (action === 'auth-nav') {
    state.authMode = target.dataset.mode || 'signin'
    location.hash = 'auth'
    render()
  }
  if (action === 'scroll-to') {
    document.querySelector(target.dataset.target || '')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
  if (action === 'signout') signOut()
  if (action === 'auth-mode') {
    state.authMode = target.dataset.mode || 'signin'
    state.error = ''
    state.message = ''
    render()
  }
  if (action === 'copy-key') copyKey(target.dataset.key || '')
  if (action === 'rotate-key') rotateKey(Number(target.dataset.id))
  if (action === 'load-models') loadModels(target.dataset.key || '')
  if (action === 'buy-plan') buyPlan(Number(target.dataset.id), Number(target.dataset.amount))
  if (action === 'buy-topup') buyTopup()
  if (action === 'verify-order') verifyOrder(target.dataset.trade || '')
})

document.addEventListener('submit', async (event) => {
  const form = event.target
  if (!(form instanceof HTMLFormElement)) return

  if (form.id === 'auth-form') {
    event.preventDefault()
    if (state.authMode === 'signin') {
      await signIn(form)
    } else {
      await register(form)
    }
  }
})

boot()

async function boot() {
  try {
    state.settings = await api('/settings/public', { auth: false })
  } catch {
    state.settings = null
  }

  if (isAuthed()) {
    try {
      state.profile = await customerApi('/user/profile')
      state.user = state.profile
      writeJson(storage.user, state.user)
    } catch {
      clearAuth()
    }
  }

  render()
  if (state.view === 'portal' && isAuthed()) {
    loadPortal()
  }
}

function render() {
  const authed = isAuthed()
  app.innerHTML = `
    <div class="site">
      <div class="shell">
        ${renderTopbar(authed)}
        ${state.view === 'auth' ? renderAuth() : ''}
        ${state.view === 'portal' ? renderPortal() : ''}
        ${state.view === 'docs' ? renderDocs() : ''}
        ${state.view === 'home' ? renderHome(authed) : ''}
        <footer class="footer">
          SpearRelay customer portal. Sub2API stays behind the operator boundary.
        </footer>
      </div>
    </div>
  `

  initInteractions()
  if (state.paymentIntent && !state.stripeMounted) {
    mountStripePaymentElement()
  }
  if (state.view === 'auth') {
    mountTurnstile()
  }
}

function renderTopbar(authed) {
  return `
    <header class="topbar">
      <button class="brand" data-action="nav" data-view="home" aria-label="SpearRelay home">
        <span class="brand-mark">SR</span>
        <span>
          <strong>SpearRelay</strong>
          <span>AI relay</span>
        </span>
      </button>
      <nav class="nav-links">
        <a href="#home">Home</a>
        <a href="#docs">Docs</a>
        ${authed ? '<a href="#portal">Portal</a>' : '<a href="#auth">Sign in</a>'}
      </nav>
      <div class="top-actions">
        ${
          authed
            ? `<button class="button button-secondary magnetic" data-action="nav" data-view="portal">Portal</button>
               <button class="button button-flat magnetic" data-action="signout">Sign out</button>`
            : `<button class="button button-secondary magnetic" data-action="auth-nav" data-mode="signin">Sign in</button>
               <button class="button button-primary magnetic" data-action="auth-nav" data-mode="signup">Start</button>`
        }
      </div>
    </header>
  `
}

function renderHome(authed) {
  return `
    <main>
      <section class="hero">
        <div class="hero-copy reveal" style="--index: 0">
          <p class="eyebrow">Private AI relay</p>
          <h1>One clean portal for buying and using API access.</h1>
          <p class="lede">Sign in. Fund balance. Copy the provisioned key. Build.</p>
          <div class="cta-row">
            <button class="button button-primary magnetic" data-action="${authed ? 'nav' : 'auth-nav'}" data-view="portal" data-mode="signup">
              ${authed ? 'Open portal' : 'Start'}
            </button>
            <button class="button button-secondary magnetic" data-action="nav" data-view="docs">API docs</button>
          </div>
          <div class="proof-grid">
            ${proof('1 key', 'Provisioned and rotatable')}
            ${proof('Models', 'Exact list for the key')}
            ${proof('Billing', 'Credits and plans')}
          </div>
        </div>

        <aside class="relay-panel glass-panel reveal" style="--index: 1" aria-label="SpearRelay product preview">
          <div class="relay-orbit" aria-hidden="true">
            <span></span><span></span><span></span>
          </div>
          <div class="relay-stack">
            <div class="relay-row active"><span>Key</span><strong>Ready</strong></div>
            <div class="relay-row"><span>Models</span><strong>Live</strong></div>
            <div class="relay-row"><span>Balance</span><strong>$100.00</strong></div>
          </div>
          <div class="tiny-code">
            <span>Authorization</span>
            <code>Bearer sk-...</code>
          </div>
        </aside>
      </section>

      <section class="section bento-section">
        <div class="section-intro reveal" style="--index: 0">
          <p class="eyebrow">Customer surface</p>
          <h2>Small surface. Clear boundary.</h2>
        </div>
        <div class="bento-grid">
          ${bentoTile('wide', 'Buy', 'Stripe-backed top-ups and plans.', 'top-up')}
          ${bentoTile('', 'Rotate', 'One provisioned key. No arbitrary key sprawl.', 'key rotation')}
          ${bentoTile('', 'Models', 'Users see only what their key can use.', 'model list')}
          ${bentoTile('tall', 'Docs', 'Messages, Chat Completions, Responses, streaming.', 'api guide')}
        </div>
      </section>

      <section class="section">
        <div class="flow-strip glass-panel reveal" style="--index: 0">
          ${flowStep('01', 'Sign up', 'Customer account only.')}
          ${flowStep('02', 'Fund', 'Credits or subscription.')}
          ${flowStep('03', 'Build', 'Use the SpearRelay key.')}
        </div>
      </section>
    </main>
  `
}

function renderAuth() {
  return `
    <main class="auth-wrap">
      <section class="auth-copy reveal" style="--index: 0">
        <p class="eyebrow">Customer account</p>
        <h1>${state.authMode === 'signin' ? 'Sign in.' : 'Create account.'}</h1>
        <p class="lede">Billing, key rotation, model list, docs. Nothing from the admin console.</p>
      </section>
      <section class="auth-card glass-panel reveal" style="--index: 1">
        <div class="auth-toggle">
          <button class="${state.authMode === 'signin' ? 'active' : ''}" data-action="auth-mode" data-mode="signin" type="button">Sign in</button>
          <button class="${state.authMode === 'signup' ? 'active' : ''}" data-action="auth-mode" data-mode="signup" type="button">Sign up</button>
        </div>
        ${state.error ? `<div class="notice">${escapeHtml(state.error)}</div>` : ''}
        ${state.message ? `<div class="notice success">${escapeHtml(state.message)}</div>` : ''}
        <form id="auth-form" class="input-row">
          <div class="field">
            <label>Email</label>
            <input name="email" type="email" autocomplete="email" required />
          </div>
          <div class="field">
            <label>Password</label>
            <input name="password" type="password" autocomplete="${state.authMode === 'signin' ? 'current-password' : 'new-password'}" required />
          </div>
          ${
            state.authMode === 'signup' && state.verifyCodeSent
              ? `<div class="field">
                  <label>Email verification code</label>
                  <input name="verify_code" inputmode="numeric" autocomplete="one-time-code" />
                </div>`
              : ''
          }
          ${renderTurnstile()}
          <button class="button button-primary magnetic" type="submit" ${state.loading ? 'disabled' : ''}>
            ${state.loading ? 'Working...' : state.authMode === 'signin' ? 'Sign in' : state.verifyCodeSent ? 'Verify and create account' : 'Create account'}
          </button>
        </form>
      </section>
    </main>
  `
}

function renderPortal() {
  if (!isAuthed()) {
    return `
      <main class="auth-wrap">
        <section class="auth-copy">
          <p class="eyebrow">Sign in required</p>
          <h1>Open the portal after sign-in.</h1>
          <p class="lede">Customer billing, key, models, orders, docs.</p>
          <button class="button button-primary magnetic" data-action="auth-nav" data-mode="signin">Sign in</button>
        </section>
      </main>
    `
  }

  const user = state.profile || state.user || {}
  return `
    <main class="portal">
      <section class="portal-head">
        <div>
          <p class="eyebrow">Customer portal</p>
          <h1>${escapeHtml(user.username || user.email || 'Your account')}</h1>
          <p class="muted">Billing, provisioned key, models, orders.</p>
        </div>
        <span class="status-pill">${escapeHtml(user.email || 'Signed in')}</span>
      </section>

      ${state.error ? `<div class="notice">${escapeHtml(state.error)}</div>` : ''}
      ${state.message ? `<div class="notice success">${escapeHtml(state.message)}</div>` : ''}

      <section class="portal-grid">
        <div>
          <div class="panel glass-panel">
            <div class="panel-title">
              <div>
                <p class="eyebrow">Balance</p>
                <h3>API spend</h3>
              </div>
              <button class="button button-secondary magnetic" data-action="scroll-to" data-target="#billing">Add credits</button>
            </div>
            <div class="balance">$${money(user.balance || 0)}</div>
            <p class="muted">Consumed by token usage at the assigned customer rate.</p>
          </div>

          <div class="panel glass-panel">
            <div class="panel-title">
              <div>
                <p class="eyebrow">API key</p>
                <h3>Provisioned access</h3>
              </div>
            </div>
            ${renderKeyPanel()}
          </div>
        </div>

        <div>
          <div class="panel glass-panel" id="billing">
            <div class="panel-title">
              <div>
                <p class="eyebrow">Billing</p>
                <h3>Packages and plans</h3>
              </div>
            </div>
            ${renderBilling()}
          </div>

          <div class="panel glass-panel">
            <div class="panel-title">
              <div>
                <p class="eyebrow">Models</p>
                <h3>Available for your key</h3>
              </div>
            </div>
            ${renderModels()}
          </div>

          <div class="panel glass-panel">
            <div class="panel-title">
              <div>
                <p class="eyebrow">Orders</p>
                <h3>Recent purchases</h3>
              </div>
            </div>
            ${renderOrders()}
          </div>
        </div>
      </section>
    </main>
  `
}

function renderKeyPanel() {
  if (state.loading && state.keys.length === 0) return skeletonStack()
  if (state.keys.length === 0) {
    return `
      <div class="key-card glass-panel">
        <p class="muted">No key is provisioned yet. Contact support and we will attach one.</p>
      </div>
    `
  }
  return state.keys.map((key) => `
    <article class="key-card glass-panel">
      <p class="tag">${escapeHtml(key.name || 'SpearRelay key')}</p>
      <code class="key-value">${escapeHtml(key.key || '')}</code>
      <div class="action-row">
        <button class="button button-secondary magnetic" data-action="copy-key" data-key="${escapeAttr(key.key || '')}">Copy</button>
        <button class="button button-flat magnetic" data-action="load-models" data-key="${escapeAttr(key.key || '')}">Models</button>
        <button class="button button-danger magnetic" data-action="rotate-key" data-id="${key.id}">Rotate</button>
      </div>
      <p class="muted">Rotation changes the secret value only.</p>
    </article>
  `).join('')
}

function renderBilling() {
  const checkout = state.checkout
  if (!checkout) return skeletonStack()

  const methods = Object.entries(checkout.methods || {})
    .filter(([, limit]) => limit && limit.available !== false)
    .map(([method]) => method)
  const defaultMethod = methods.includes('stripe') ? 'stripe' : methods[0]

  return `
    <div class="billing-grid">
      <article class="plan-card glass-panel">
        <p class="tag">Top-up</p>
        <h3>Add credits</h3>
        <p class="muted">For token-metered calls.</p>
        <div class="field">
          <label>Amount</label>
          <input id="topup-amount" type="number" min="${checkout.global_min || 1}" max="${checkout.global_max || 5000}" step="1" value="50" />
        </div>
        <div class="field">
          <label>Payment method</label>
          <select id="topup-method">
            ${methods.map((method) => `<option value="${escapeAttr(method)}" ${method === defaultMethod ? 'selected' : ''}>${escapeHtml(labelPayment(method))}</option>`).join('')}
          </select>
        </div>
        <button class="button button-primary magnetic" data-action="buy-topup" ${methods.length === 0 ? 'disabled' : ''}>Buy credits</button>
      </article>
      ${(checkout.plans || []).map((plan) => `
        <article class="plan-card glass-panel">
          <p class="tag">Subscription</p>
          <h3>${escapeHtml(plan.name)}</h3>
          <div class="plan-price">${currency(plan.price)}</div>
          <p class="muted">${escapeHtml(plan.description || 'Recurring API access plan.')}</p>
          <button class="button button-primary magnetic" data-action="buy-plan" data-id="${plan.id}" data-amount="${Number(plan.price || 0)}" ${methods.length === 0 ? 'disabled' : ''}>
            Subscribe
          </button>
        </article>
      `).join('')}
    </div>
    ${state.paymentIntent ? renderStripeIntent() : ''}
  `
}

function renderModels() {
  if (state.models.length === 0) {
    return '<p class="muted">Click Models on your key to fetch the exact list.</p>'
  }
  return `
    <div class="model-grid">
      ${state.models.map((model) => `
        <article class="model-card glass-panel">
          <p class="tag">Model</p>
          <h3>${escapeHtml(model.id || model.name || 'unknown-model')}</h3>
          <p class="muted">${escapeHtml(model.owned_by || 'spearrelay')}</p>
        </article>
      `).join('')}
    </div>
  `
}

function renderOrders() {
  if (!state.orders || state.orders.length === 0) return '<p class="muted">No recent orders yet.</p>'
  return `
    <div class="orders-grid">
      ${state.orders.map((order) => `
        <article class="order-card glass-panel">
          <p class="tag">${escapeHtml(order.status || 'PENDING')}</p>
          <h3>${currency(order.pay_amount || order.amount || 0)}</h3>
          <p>Type: ${escapeHtml(order.order_type || 'balance')} · Method: ${escapeHtml(order.payment_type || '-')}</p>
          <p>Order: ${escapeHtml(order.out_trade_no || String(order.id))}</p>
          ${
            order.status === 'PENDING' && order.out_trade_no
              ? `<button class="button button-secondary magnetic" data-action="verify-order" data-trade="${escapeAttr(order.out_trade_no)}">Refresh</button>`
              : ''
          }
        </article>
      `).join('')}
    </div>
  `
}

function renderStripeIntent() {
  return `
    <div class="panel glass-panel">
      <div class="panel-title">
        <div>
          <p class="eyebrow">Stripe checkout</p>
          <h3>Complete payment</h3>
        </div>
      </div>
      <div id="stripe-payment-element"></div>
      <div class="action-row">
        <button id="stripe-submit" class="button button-primary magnetic" type="button">Pay now</button>
      </div>
      <p class="muted">Customer purchase amount only.</p>
    </div>
  `
}

function renderDocs() {
  return `
    <main class="docs-shell">
      <p class="eyebrow">API documentation</p>
      <h1>Use your provisioned key.</h1>
      <p class="lede">Keep keys out of browser code, public repos, URLs, and logs.</p>

      ${docBlock('Base URL and authentication', `export SPEARRELAY_BASE_URL="${gatewayBaseUrl()}"\nexport SPEARRELAY_API_KEY="sk-your-provisioned-key"\n\nAuthorization: Bearer $SPEARRELAY_API_KEY`)}
      ${docBlock('List models', `curl -sS "$SPEARRELAY_BASE_URL/v1/models" \\\n  -H "Authorization: Bearer $SPEARRELAY_API_KEY"`)}
      ${docBlock('Anthropic Messages API', `curl -sS "$SPEARRELAY_BASE_URL/v1/messages" \\\n  -H "Authorization: Bearer $SPEARRELAY_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -H "anthropic-version: 2023-06-01" \\\n  -d '{\n    "model": "claude-sonnet-4-6",\n    "max_tokens": 800,\n    "messages": [\n      { "role": "user", "content": "Reply with a short answer." }\n    ]\n  }'`)}
      ${docBlock('OpenAI Chat Completions API', `curl -sS "$SPEARRELAY_BASE_URL/v1/chat/completions" \\\n  -H "Authorization: Bearer $SPEARRELAY_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "claude-sonnet-4-6",\n    "messages": [\n      { "role": "user", "content": "Compare these two options." }\n    ],\n    "reasoning_effort": "high"\n  }'`)}
      ${docBlock('OpenAI Responses API', `curl -sS "$SPEARRELAY_BASE_URL/v1/responses" \\\n  -H "Authorization: Bearer $SPEARRELAY_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "claude-sonnet-4-6",\n    "input": "Write a concise implementation plan.",\n    "reasoning": { "effort": "high" }\n  }'`)}
      ${docBlock('Streaming', `{\n  "model": "claude-sonnet-4-6",\n  "max_tokens": 1200,\n  "stream": true,\n  "messages": [\n    { "role": "user", "content": "Draft a migration plan." }\n  ]\n}`)}
    </main>
  `
}

function proof(value, label) {
  return `<div class="proof-item"><strong>${escapeHtml(value)}</strong><span>${escapeHtml(label)}</span></div>`
}

function bentoTile(size, title, body, label) {
  return `
    <article class="bento-tile glass-panel ${escapeAttr(size)} reveal">
      <span class="card-kicker">${escapeHtml(label)}</span>
      <h3>${escapeHtml(title)}</h3>
      <p>${escapeHtml(body)}</p>
      <div class="bento-visual" aria-hidden="true">${pulseRows()}</div>
    </article>
  `
}

function flowStep(number, title, body) {
  return `<article class="flow-step"><span>${escapeHtml(number)}</span><h3>${escapeHtml(title)}</h3><p>${escapeHtml(body)}</p></article>`
}

function docBlock(title, code) {
  return `<section class="doc-block glass-panel"><h3>${escapeHtml(title)}</h3><pre><code>${escapeHtml(code)}</code></pre></section>`
}

function pulseRows() {
  return '<span></span><span></span><span></span>'
}

function skeletonStack() {
  return '<div class="skeleton-stack"><span></span><span></span><span></span></div>'
}

function initInteractions() {
  document.querySelectorAll('.magnetic').forEach((element) => {
    element.addEventListener('pointermove', (event) => {
      const rect = element.getBoundingClientRect()
      const x = (event.clientX - rect.left - rect.width / 2) / rect.width
      const y = (event.clientY - rect.top - rect.height / 2) / rect.height
      element.style.setProperty('--mx', `${x * 10}px`)
      element.style.setProperty('--my', `${y * 7}px`)
    })
    element.addEventListener('pointerleave', () => {
      element.style.setProperty('--mx', '0px')
      element.style.setProperty('--my', '0px')
    })
  })

  document.querySelectorAll('.glass-panel').forEach((element) => {
    element.addEventListener('pointermove', (event) => {
      const rect = element.getBoundingClientRect()
      element.style.setProperty('--spot-x', `${event.clientX - rect.left}px`)
      element.style.setProperty('--spot-y', `${event.clientY - rect.top}px`)
    })
  })
}

async function signIn(form) {
  await withLoading(async () => {
    const payload = formPayload(form)
    attachTurnstileToken(payload)
    const result = await customerApi('/auth/login', { method: 'POST', body: payload, auth: false })
    saveAuth(result)
    state.message = 'Signed in. Opening your customer portal.'
    location.hash = 'portal'
    await loadPortal()
  })
}

async function register(form) {
  await withLoading(async () => {
    const payload = formPayload(form)
    attachTurnstileToken(payload)
    if (state.settings?.email_verify_enabled && !payload.verify_code) {
      await customerApi('/auth/send-verify-code', {
        method: 'POST',
        body: { email: payload.email, turnstile_token: payload.turnstile_token },
        auth: false
      })
      state.verifyCodeSent = true
      state.message = 'Verification code sent. Enter the code to create your account.'
      render()
      return
    }
    const result = await customerApi('/auth/register', { method: 'POST', body: payload, auth: false })
    saveAuth(result)
    state.message = 'Account created. Opening your customer portal.'
    location.hash = 'portal'
    await loadPortal()
  })
}

async function loadPortal() {
  if (!isAuthed()) return
  await withLoading(async () => {
    const results = await Promise.allSettled([
      customerApi('/user/profile'),
      customerApi('/keys?page=1&page_size=10'),
      customerApi('/payment/checkout-info'),
      customerApi('/payment/orders/my?page=1&page_size=8'),
      customerApi('/subscriptions/summary')
    ])

    if (results[0].status === 'fulfilled') {
      state.profile = results[0].value
      state.user = state.profile
      writeJson(storage.user, state.user)
    }
    if (results[1].status === 'fulfilled') state.keys = results[1].value.items || []
    if (results[2].status === 'fulfilled') state.checkout = results[2].value
    if (results[3].status === 'fulfilled') state.orders = results[3].value.items || []
    if (results[4].status === 'fulfilled') state.subscriptions = results[4].value
  }, { rerender: false })
  render()
}

async function loadModels(apiKey) {
  if (!apiKey) return
  await withLoading(async () => {
    const response = await fetch(`${gatewayBaseUrl()}/v1/models`, {
      headers: { Authorization: `Bearer ${apiKey}` }
    })
    if (!response.ok) throw new Error(`Model lookup failed: HTTP ${response.status}`)
    const data = await response.json()
    state.models = Array.isArray(data.data) ? data.data : []
    state.message = `Loaded ${state.models.length} models for this key.`
  })
}

async function rotateKey(id) {
  if (!id || !confirm('Rotate this API key now? Existing clients using the old key will stop working.')) return
  await withLoading(async () => {
    const rotated = await customerApi(`/keys/${id}/rotate`, { method: 'POST' })
    state.keys = state.keys.map((key) => (key.id === id ? rotated : key))
    state.message = 'API key rotated. Update your clients with the new key.'
  })
}

async function buyTopup() {
  const amount = Number(document.querySelector('#topup-amount')?.value || 0)
  const method = document.querySelector('#topup-method')?.value || 'stripe'
  if (!amount || amount <= 0) {
    state.error = 'Enter a valid top-up amount.'
    render()
    return
  }
  await createOrder({ amount, payment_type: method, order_type: 'balance' })
}

async function buyPlan(planId, amount) {
  const methods = Object.keys(state.checkout?.methods || {})
  const method = methods.includes('stripe') ? 'stripe' : methods[0]
  if (!method) {
    state.error = 'No payment method is currently available.'
    render()
    return
  }
  await createOrder({ amount, payment_type: method, order_type: 'subscription', plan_id: planId })
}

async function createOrder(payload) {
  await withLoading(async () => {
    const result = await customerApi('/payment/orders', {
      method: 'POST',
      body: {
        ...payload,
        return_url: `${location.origin}${location.pathname}#portal`,
        payment_source: 'spearrelay'
      }
    })

    if (result.pay_url) {
      location.href = result.pay_url
      return
    }
    if (result.client_secret) {
      state.paymentIntent = {
        clientSecret: result.client_secret,
        publishableKey: state.checkout?.stripe_publishable_key,
        orderId: result.order_id
      }
      state.stripeMounted = false
      state.message = 'Stripe payment created. Complete payment below.'
      render()
      return
    }
    state.message = 'Order created. Check recent orders for status.'
    await loadPortal()
  })
}

async function verifyOrder(outTradeNo) {
  if (!outTradeNo) return
  await withLoading(async () => {
    await customerApi('/payment/orders/verify', { method: 'POST', body: { out_trade_no: outTradeNo } })
    state.message = 'Order status refreshed.'
    await loadPortal()
  })
}

async function mountStripePaymentElement() {
  if (!window.Stripe || !state.paymentIntent?.publishableKey || !state.paymentIntent?.clientSecret) return

  state.stripeMounted = true
  const stripe = window.Stripe(state.paymentIntent.publishableKey)
  const elements = stripe.elements({ clientSecret: state.paymentIntent.clientSecret })
  const paymentElement = elements.create('payment')
  paymentElement.mount('#stripe-payment-element')
  document.querySelector('#stripe-submit')?.addEventListener('click', async () => {
    const button = document.querySelector('#stripe-submit')
    button.disabled = true
    const { error } = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: `${location.origin}${location.pathname}#portal`
      }
    })
    if (error) {
      state.error = error.message || 'Stripe payment failed.'
      button.disabled = false
      render()
    }
  })
}

function renderTurnstile() {
  if (!state.settings?.turnstile_enabled || !state.settings?.turnstile_site_key) return ''
  return `
    <div class="field">
      <label>Security check</label>
      <div id="turnstile-container" class="turnstile-box"></div>
    </div>
  `
}

function attachTurnstileToken(payload) {
  if (state.settings?.turnstile_enabled) {
    payload.turnstile_token = state.turnstileToken
  }
}

function mountTurnstile() {
  if (!state.settings?.turnstile_enabled || !state.settings?.turnstile_site_key) return
  const container = document.querySelector('#turnstile-container')
  if (!container || container.dataset.mounted === '1') return

  loadTurnstileScript().then(() => {
    if (!window.turnstile || !document.querySelector('#turnstile-container')) return
    state.turnstileToken = ''
    container.dataset.mounted = '1'
    state.turnstileWidgetId = window.turnstile.render(container, {
      sitekey: state.settings.turnstile_site_key,
      callback(token) {
        state.turnstileToken = token
      },
      'expired-callback'() {
        state.turnstileToken = ''
      },
      'error-callback'() {
        state.turnstileToken = ''
      }
    })
  })
}

function loadTurnstileScript() {
  if (window.turnstile) return Promise.resolve()
  const existing = document.querySelector('script[data-turnstile]')
  if (existing) {
    return new Promise((resolve) => existing.addEventListener('load', resolve, { once: true }))
  }
  return new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
    script.async = true
    script.defer = true
    script.dataset.turnstile = '1'
    script.onload = resolve
    script.onerror = reject
    document.head.appendChild(script)
  })
}

async function api(path, options = {}) {
  const auth = options.auth !== false
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {})
  }
  const token = localStorage.getItem(storage.access)
  if (auth && token) headers.Authorization = `Bearer ${token}`

  const response = await fetch(`${apiBaseUrl()}${path}`, {
    method: options.method || 'GET',
    headers,
    credentials: 'include',
    body: options.body ? JSON.stringify(options.body) : undefined
  })

  if (response.status === 401 && auth && localStorage.getItem(storage.refresh)) {
    const refreshed = await refreshAccessToken()
    if (refreshed) return api(path, options)
  }

  const raw = await safeJson(response)
  if (!response.ok) {
    throw new Error(extractError(raw) || `HTTP ${response.status}`)
  }
  if (raw && typeof raw === 'object' && 'code' in raw) {
    if (raw.code === 0) return raw.data
    throw new Error(raw.message || 'Request failed')
  }
  return raw
}

async function customerApi(path, options = {}) {
  return api(`/customer${path}`, options)
}

async function refreshAccessToken() {
  try {
    const refreshToken = localStorage.getItem(storage.refresh)
    if (!refreshToken) return false
    const result = await customerApi('/auth/refresh', {
      method: 'POST',
      body: { refresh_token: refreshToken },
      auth: false
    })
    localStorage.setItem(storage.access, result.access_token)
    localStorage.setItem(storage.refresh, result.refresh_token)
    return true
  } catch {
    clearAuth()
    return false
  }
}

async function withLoading(fn, options = {}) {
  state.loading = true
  state.error = ''
  if (options.rerender !== false) render()
  try {
    await fn()
  } catch (error) {
    state.error = error.message || 'Something went wrong.'
  } finally {
    state.loading = false
    if (options.rerender !== false) render()
  }
}

function saveAuth(result) {
  localStorage.setItem(storage.access, result.access_token)
  localStorage.setItem(storage.refresh, result.refresh_token)
  state.user = result.user
  state.profile = result.user
  writeJson(storage.user, result.user)
}

function signOut() {
  clearAuth()
  state.profile = null
  state.keys = []
  state.models = []
  state.checkout = null
  state.orders = []
  location.hash = 'home'
  render()
}

function clearAuth() {
  localStorage.removeItem(storage.access)
  localStorage.removeItem(storage.refresh)
  localStorage.removeItem(storage.user)
}

function isAuthed() {
  return Boolean(localStorage.getItem(storage.access))
}

function apiBaseUrl() {
  return (localStorage.getItem(storage.apiBase) || config.apiBaseUrl || '/api/v1').replace(/\/$/, '')
}

function gatewayBaseUrl() {
  if (config.gatewayBaseUrl) return config.gatewayBaseUrl.replace(/\/$/, '')
  return apiBaseUrl().replace(/\/api\/v1$/, '')
}

function formPayload(form) {
  return Object.fromEntries(new FormData(form).entries())
}

function readJson(key) {
  try {
    return JSON.parse(localStorage.getItem(key) || 'null')
  } catch {
    return null
  }
}

function writeJson(key, value) {
  localStorage.setItem(key, JSON.stringify(value))
}

async function safeJson(response) {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return { message: text }
  }
}

function extractError(raw) {
  return raw?.message || raw?.detail || raw?.error?.message || ''
}

function copyKey(value) {
  navigator.clipboard?.writeText(value)
  state.message = 'API key copied.'
  render()
}

function labelPayment(method) {
  return {
    stripe: 'Stripe',
    alipay: 'Alipay',
    wxpay: 'WeChat Pay',
    easypay: 'EasyPay',
    alipay_direct: 'Alipay',
    wxpay_direct: 'WeChat Pay'
  }[method] || method
}

function money(value) {
  return Number(value || 0).toFixed(2)
}

function currency(value) {
  return `$${money(value)}`
}

function escapeHtml(value) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

function escapeAttr(value) {
  return escapeHtml(value).replace(/`/g, '&#096;')
}
