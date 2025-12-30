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

func CreateCheckoutSession(inviteCode string, amount int64, currency, successURL, cancelURL string, metadata map[string]string, interval string) (string, error) {
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
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(inviteCode),
	}

	if interval != "" {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModeSubscription))
		params.LineItems[0].PriceData.Recurring = &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
			Interval: stripe.String(interval),
		}
	} else {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModePayment))
	}

	if metadata != nil {
		params.Metadata = metadata
	}

	s, err := session.New(params)
	if err != nil {
		return "", err
	}

	return s.URL, nil
}

func HandleWebhook(payload []byte, signature string, secret string, verifySignature bool) (*stripe.Event, error) {
	var event stripe.Event
	var err error

	if verifySignature {
		event, err = webhook.ConstructEvent(payload, signature, secret)
		if err != nil {
			return nil, fmt.Errorf("bad_signature: %w", err)
		}
	} else {
		// Bypass Signature: Use explicit API Call-Back to verify event authenticity.
		var untrustedEvent stripe.Event
		if err := json.Unmarshal(payload, &untrustedEvent); err != nil {
			return nil, fmt.Errorf("webhook_json_parse_error: %w", err)
		}

		eventPtr, err := stripeEvent.Get(untrustedEvent.ID, nil)
		if err != nil {
			return nil, fmt.Errorf("api_verification_failed: %w", err)
		}
		event = *eventPtr
	}

	return &event, nil
}
