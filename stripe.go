package main

import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	stripeEvent "github.com/stripe/stripe-go/v79/event"
	"github.com/stripe/stripe-go/v79/webhook"
)

func InitStripe(apiKey string) {
	stripe.Key = apiKey
}

func CreateCheckoutSession(inviteCode string, amount int64, currency, successURL, cancelURL string) (string, error) {
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Invite Code: " + inviteCode),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(inviteCode),
	}

	s, err := session.New(params)
	if err != nil {
		return "", err
	}

	return s.URL, nil
}

func HandleWebhook(payload []byte, signature string, secret string, verifySignature bool) (string, error) {
	var event stripe.Event
	var err error

	if verifySignature {
		event, err = webhook.ConstructEvent(payload, signature, secret)
		if err != nil {
			return "", fmt.Errorf("bad_signature: %w", err)
		}
	} else {
		// Bypass Signature: Use explicit API Call-Back to verify event authenticity.
		// 1. Unmarshal payload just to get the Event ID
		var untrustedEvent stripe.Event
		if err := json.Unmarshal(payload, &untrustedEvent); err != nil {
			return "", fmt.Errorf("webhook_json_parse_error: %w", err)
		}

		// 2. Call Stripe API to get the authoritative Event object
		// This protects against spoofed payloads since we trust only what Stripe's API returns.
		eventPtr, err := stripeEvent.Get(untrustedEvent.ID, nil)
		if err != nil {
			return "", fmt.Errorf("api_verification_failed: %w", err)
		}
		event = *eventPtr
	}

	if event.Type == "checkout.session.completed" {
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return "", fmt.Errorf("parse_error")
		}

		if session.ClientReferenceID != "" {
			return session.ClientReferenceID, nil
		}
	}

	return "", nil
}
