package main

import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/checkout/session"
	stripeEvent "github.com/stripe/stripe-go/v86/event"
	"github.com/stripe/stripe-go/v86/webhook"
)

const supportedStripeWebhookAPIVersion = "2026-06-24.dahlia"

func InitStripe(apiKey string) {
	stripe.Key = apiKey
}

func CreateCheckoutSession(inviteCode string, amount int64, currency, productName, successURL, cancelURL string, metadata map[string]string, interval string, intervalCount int64) (*stripe.CheckoutSession, error) {
	if productName == "" {
		productName = "Invite Code: " + inviteCode
	}
	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(productName),
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
		if intervalCount > 1 {
			params.LineItems[0].PriceData.Recurring.IntervalCount = stripe.Int64(intervalCount)
		}
	} else {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModePayment))
	}

	if metadata != nil {
		params.Metadata = metadata
		if interval != "" {
			params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
				Metadata: metadata,
			}
		}
	}

	s, err := session.New(params)
	if err != nil {
		return nil, err
	}

	return s, nil
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

	if event.APIVersion != supportedStripeWebhookAPIVersion {
		return nil, fmt.Errorf("Stripe webhook API version %q does not match supported version %q", event.APIVersion, supportedStripeWebhookAPIVersion)
	}

	return &event, nil
}
