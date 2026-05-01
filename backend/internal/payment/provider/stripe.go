package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	stripe "github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
)

// Stripe constants.
const (
	stripeCurrency                      = "cny"
	stripeEventPaymentSuccess           = "payment_intent.succeeded"
	stripeEventPaymentFailed            = "payment_intent.payment_failed"
	stripeEventCheckoutSessionCompleted = "checkout.session.completed"
	stripeEventInvoicePaid              = "invoice.paid"
	stripeEventInvoicePaymentFailed     = "invoice.payment_failed"
)

// Stripe implements the payment.CancelableProvider interface for Stripe payments.
type Stripe struct {
	instanceID string
	config     map[string]string

	mu          sync.Mutex
	initialized bool
	sc          *stripe.Client
}

// NewStripe creates a new Stripe provider instance.
func NewStripe(instanceID string, config map[string]string) (*Stripe, error) {
	if config["secretKey"] == "" {
		return nil, fmt.Errorf("stripe config missing required key: secretKey")
	}
	if config["publishableKey"] == "" {
		return nil, fmt.Errorf("stripe config missing required key: publishableKey")
	}
	if config["webhookSecret"] == "" {
		return nil, fmt.Errorf("stripe config missing required key: webhookSecret")
	}
	return &Stripe{
		instanceID: instanceID,
		config:     config,
	}, nil
}

func (s *Stripe) ensureInit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.initialized {
		s.sc = stripe.NewClient(s.config["secretKey"])
		s.initialized = true
	}
}

// GetPublishableKey returns the publishable key for frontend use.
func (s *Stripe) GetPublishableKey() string {
	return s.config["publishableKey"]
}

func (s *Stripe) Name() string        { return "Stripe" }
func (s *Stripe) ProviderKey() string { return payment.TypeStripe }
func (s *Stripe) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeStripe}
}

// stripePaymentMethodTypes maps our PaymentType to Stripe payment_method_types.
var stripePaymentMethodTypes = map[string][]string{
	payment.TypeCard:   {"card"},
	payment.TypeAlipay: {"alipay"},
	payment.TypeWxpay:  {"wechat_pay"},
	payment.TypeLink:   {"link"},
}

// CreatePayment creates a Stripe PaymentIntent.
func (s *Stripe) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	s.ensureInit()

	if strings.EqualFold(strings.TrimSpace(req.OrderType), payment.OrderTypeSubscription) && strings.TrimSpace(req.StripePriceID) != "" {
		return s.createSubscriptionCheckout(ctx, req)
	}

	amountInCents, err := payment.YuanToFen(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("stripe create payment: %w", err)
	}

	// Collect all Stripe payment_method_types from the instance's configured sub-methods
	methods := resolveStripeMethodTypes(req.InstanceSubMethods)

	pmTypes := make([]*string, len(methods))
	for i, m := range methods {
		pmTypes[i] = stripe.String(m)
	}

	params := &stripe.PaymentIntentCreateParams{
		Amount:             stripe.Int64(amountInCents),
		Currency:           stripe.String(stripeCurrency),
		PaymentMethodTypes: pmTypes,
		Description:        stripe.String(req.Subject),
		Metadata:           map[string]string{"orderId": req.OrderID},
	}

	// WeChat Pay requires payment_method_options with client type
	if hasStripeMethod(methods, "wechat_pay") {
		params.PaymentMethodOptions = &stripe.PaymentIntentCreatePaymentMethodOptionsParams{
			WeChatPay: &stripe.PaymentIntentCreatePaymentMethodOptionsWeChatPayParams{
				Client: stripe.String("web"),
			},
		}
	}

	params.SetIdempotencyKey(fmt.Sprintf("pi-%s", req.OrderID))
	params.Context = ctx

	pi, err := s.sc.V1PaymentIntents.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("stripe create payment: %w", err)
	}

	return &payment.CreatePaymentResponse{
		TradeNo:      pi.ID,
		ClientSecret: pi.ClientSecret,
	}, nil
}

func (s *Stripe) createSubscriptionCheckout(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	metadata := map[string]string{
		"orderId":   req.OrderID,
		"orderType": payment.OrderTypeSubscription,
	}
	if req.PlanID > 0 {
		metadata["planId"] = fmt.Sprintf("%d", req.PlanID)
	}

	successURL := strings.TrimSpace(req.ReturnURL)
	cancelURL := strings.TrimSpace(req.ReturnURL)
	if successURL == "" {
		successURL = strings.TrimSpace(s.config["successUrl"])
	}
	if cancelURL == "" {
		cancelURL = strings.TrimSpace(s.config["cancelUrl"])
	}

	params := &stripe.CheckoutSessionCreateParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ClientReferenceID: stripe.String(req.OrderID),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripe.String(strings.TrimSpace(req.StripePriceID)),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: metadata,
		SubscriptionData: &stripe.CheckoutSessionCreateSubscriptionDataParams{
			Metadata: metadata,
		},
	}
	if successURL != "" {
		params.SuccessURL = stripe.String(successURL)
	}
	if cancelURL != "" {
		params.CancelURL = stripe.String(cancelURL)
	}
	params.SetIdempotencyKey(fmt.Sprintf("checkout-sub-%s", req.OrderID))
	params.Context = ctx

	session, err := s.sc.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("stripe create subscription checkout: %w", err)
	}
	return &payment.CreatePaymentResponse{
		TradeNo: session.ID,
		PayURL:  session.URL,
	}, nil
}

// QueryOrder retrieves a PaymentIntent by ID.
func (s *Stripe) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	s.ensureInit()

	pi, err := s.sc.V1PaymentIntents.Retrieve(ctx, tradeNo, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe query order: %w", err)
	}

	status := payment.ProviderStatusPending
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		status = payment.ProviderStatusPaid
	case stripe.PaymentIntentStatusCanceled:
		status = payment.ProviderStatusFailed
	}

	return &payment.QueryOrderResponse{
		TradeNo: pi.ID,
		Status:  status,
		Amount:  payment.FenToYuan(pi.Amount),
	}, nil
}

// VerifyNotification verifies a Stripe webhook event.
func (s *Stripe) VerifyNotification(_ context.Context, rawBody string, headers map[string]string) (*payment.PaymentNotification, error) {
	s.ensureInit()

	webhookSecret := s.config["webhookSecret"]
	if webhookSecret == "" {
		return nil, fmt.Errorf("stripe webhookSecret not configured")
	}

	sig := headers["stripe-signature"]
	if sig == "" {
		return nil, fmt.Errorf("stripe notification missing stripe-signature header")
	}

	event, err := webhook.ConstructEvent([]byte(rawBody), sig, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("stripe verify notification: %w", err)
	}

	switch event.Type {
	case stripeEventPaymentSuccess:
		return parseStripePaymentIntent(&event, payment.ProviderStatusSuccess, rawBody)
	case stripeEventPaymentFailed:
		return parseStripePaymentIntent(&event, payment.ProviderStatusFailed, rawBody)
	case stripeEventCheckoutSessionCompleted:
		return parseStripeCheckoutSession(&event, payment.ProviderStatusSuccess, rawBody)
	case stripeEventInvoicePaid:
		return parseStripeInvoice(&event, payment.ProviderStatusSuccess, rawBody)
	case stripeEventInvoicePaymentFailed:
		return parseStripeInvoice(&event, payment.ProviderStatusFailed, rawBody)
	}

	return nil, nil
}

func parseStripeCheckoutSession(event *stripe.Event, status string, rawBody string) (*payment.PaymentNotification, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return nil, fmt.Errorf("stripe parse checkout.session: %w", err)
	}
	orderID := session.Metadata["orderId"]
	if strings.TrimSpace(orderID) == "" {
		orderID = session.ClientReferenceID
	}
	if strings.TrimSpace(orderID) == "" {
		return nil, nil
	}
	return &payment.PaymentNotification{
		TradeNo:  session.ID,
		OrderID:  orderID,
		Amount:   payment.FenToYuan(session.AmountTotal),
		Status:   status,
		RawData:  rawBody,
		Metadata: cloneStripeStringMap(session.Metadata),
	}, nil
}

func parseStripeInvoice(event *stripe.Event, status string, rawBody string) (*payment.PaymentNotification, error) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return nil, fmt.Errorf("stripe parse invoice: %w", err)
	}
	metadata := cloneStripeStringMap(invoice.Metadata)
	if invoice.Parent != nil && invoice.Parent.SubscriptionDetails != nil {
		for k, v := range invoice.Parent.SubscriptionDetails.Metadata {
			if _, exists := metadata[k]; !exists {
				metadata[k] = v
			}
		}
	}
	orderID := metadata["orderId"]
	if strings.TrimSpace(orderID) == "" {
		return nil, nil
	}
	return &payment.PaymentNotification{
		TradeNo:  invoice.ID,
		OrderID:  orderID,
		Amount:   payment.FenToYuan(invoice.AmountPaid),
		Status:   status,
		RawData:  rawBody,
		Metadata: metadata,
	}, nil
}

func cloneStripeStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func parseStripePaymentIntent(event *stripe.Event, status string, rawBody string) (*payment.PaymentNotification, error) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		return nil, fmt.Errorf("stripe parse payment_intent: %w", err)
	}
	return &payment.PaymentNotification{
		TradeNo: pi.ID,
		OrderID: pi.Metadata["orderId"],
		Amount:  payment.FenToYuan(pi.Amount),
		Status:  status,
		RawData: rawBody,
	}, nil
}

// Refund creates a Stripe refund.
func (s *Stripe) Refund(ctx context.Context, req payment.RefundRequest) (*payment.RefundResponse, error) {
	s.ensureInit()

	amountInCents, err := payment.YuanToFen(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("stripe refund: %w", err)
	}

	params := &stripe.RefundCreateParams{
		PaymentIntent: stripe.String(req.TradeNo),
		Amount:        stripe.Int64(amountInCents),
		Reason:        stripe.String(string(stripe.RefundReasonRequestedByCustomer)),
	}
	params.Context = ctx

	r, err := s.sc.V1Refunds.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("stripe refund: %w", err)
	}

	refundStatus := payment.ProviderStatusPending
	if r.Status == stripe.RefundStatusSucceeded {
		refundStatus = payment.ProviderStatusSuccess
	}

	return &payment.RefundResponse{
		RefundID: r.ID,
		Status:   refundStatus,
	}, nil
}

// resolveStripeMethodTypes converts instance supported_types (comma-separated)
// into Stripe API payment_method_types. Falls back to ["card"] if empty.
func resolveStripeMethodTypes(instanceSubMethods string) []string {
	if instanceSubMethods == "" {
		return []string{"card"}
	}
	var methods []string
	for _, t := range strings.Split(instanceSubMethods, ",") {
		t = strings.TrimSpace(t)
		if mapped, ok := stripePaymentMethodTypes[t]; ok {
			methods = append(methods, mapped...)
		}
	}
	if len(methods) == 0 {
		return []string{"card"}
	}
	return methods
}

// hasStripeMethod checks if the given Stripe method list contains the target method.
func hasStripeMethod(methods []string, target string) bool {
	for _, m := range methods {
		if m == target {
			return true
		}
	}
	return false
}

// CancelPayment cancels a pending PaymentIntent.
func (s *Stripe) CancelPayment(ctx context.Context, tradeNo string) error {
	s.ensureInit()

	_, err := s.sc.V1PaymentIntents.Cancel(ctx, tradeNo, nil)
	if err != nil {
		return fmt.Errorf("stripe cancel payment: %w", err)
	}
	return nil
}

// Ensure interface compliance.
var (
	_ payment.Provider           = (*Stripe)(nil)
	_ payment.CancelableProvider = (*Stripe)(nil)
)
