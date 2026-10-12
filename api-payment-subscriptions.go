package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/stripe/stripe-go/v86"
	chargeapi "github.com/stripe/stripe-go/v86/charge"
	refundapi "github.com/stripe/stripe-go/v86/refund"
	subscriptionapi "github.com/stripe/stripe-go/v86/subscription"
)

const (
	paymentSubscriptionCancelNow       = "now"
	paymentSubscriptionCancelPeriodEnd = "period_end"
	paymentSubscriptionCancelCustom    = "custom"
)

func (app *appContext) CancelPaymentSubscription(gc *gin.Context) {
	if !stripeEnabled {
		respond(400, "Stripe disabled", gc)
		return
	}

	paymentID := gc.Param("id")
	payment, ok := app.storage.GetPaymentKey(paymentID)
	if !ok {
		respond(404, "Payment not found", gc)
		return
	}
	if payment.Provider != lm.Stripe || payment.SubscriptionID == "" {
		respond(400, "Payment has no Stripe subscription", gc)
		return
	}
	if payment.SubscriptionStatus == string(stripe.SubscriptionStatusCanceled) || payment.Status == paymentStatusSubscriptionCanceled {
		respond(400, "Subscription is already canceled", gc)
		return
	}

	var req cancelPaymentSubscriptionDTO
	if err := gc.ShouldBindJSON(&req); err != nil {
		respond(400, "Invalid request: "+err.Error(), gc)
		return
	}
	if req.When == "" {
		req.When = paymentSubscriptionCancelPeriodEnd
	}

	sub, err := app.cancelStripeSubscription(payment.SubscriptionID, req)
	if err != nil {
		app.err.Printf("Failed to cancel Stripe subscription %s from payment %s: %v", payment.SubscriptionID, payment.ID, err)
		respond(500, "Failed to cancel subscription", gc)
		return
	}

	app.applyStripeSubscriptionEvent(sub)
	app.notifyStripeSubscriptionCancellation(sub, "admin")
	if req.When == paymentSubscriptionCancelNow && sub.Status == stripe.SubscriptionStatusCanceled {
		app.expireStripeSubscriptionUser(sub)
	}

	refundID := ""
	if req.Refund {
		refund, err := app.refundPaymentLatestPayment(payment)
		if err != nil {
			app.err.Printf("Failed to refund Stripe payment %s while canceling subscription %s: %v", payment.ID, payment.SubscriptionID, err)
			respond(500, "Subscription canceled, but refund failed: "+err.Error(), gc)
			return
		}
		if refund != nil {
			refundID = refund.ID
		}
	}

	updated, _ := app.storage.GetPaymentKey(payment.ID)
	gc.JSON(200, cancelPaymentSubscriptionResponseDTO{
		Payment:        paymentToDTO(updated),
		SubscriptionID: sub.ID,
		RefundID:       refundID,
	})
}

func (app *appContext) cancelStripeSubscription(subscriptionID string, req cancelPaymentSubscriptionDTO) (*stripe.Subscription, error) {
	switch req.When {
	case paymentSubscriptionCancelNow:
		return subscriptionapi.Cancel(subscriptionID, &stripe.SubscriptionCancelParams{
			InvoiceNow: stripe.Bool(false),
			Prorate:    stripe.Bool(false),
		})
	case paymentSubscriptionCancelPeriodEnd:
		return subscriptionapi.Update(subscriptionID, &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		})
	case paymentSubscriptionCancelCustom:
		if req.CancelAt <= time.Now().Unix() {
			return nil, fmt.Errorf("custom cancellation date must be in the future")
		}
		return subscriptionapi.Update(subscriptionID, &stripe.SubscriptionParams{
			CancelAt:            stripe.Int64(req.CancelAt),
			CancelAtPeriodEnd:   stripe.Bool(false),
			ProrationBehavior:   stripe.String("none"),
			CancellationDetails: &stripe.SubscriptionCancellationDetailsParams{},
		})
	default:
		return nil, fmt.Errorf("invalid cancellation timing %q", req.When)
	}
}

func (app *appContext) refundPaymentLatestPayment(payment Payment) (*stripe.Refund, error) {
	payment = app.latestRefundablePayment(payment)
	remaining := payment.Amount - payment.RefundedAmount
	if remaining <= 0 {
		return nil, fmt.Errorf("payment is already fully refunded")
	}

	params := &stripe.RefundParams{
		Amount: stripe.Int64(remaining),
		Reason: stripe.String(string(stripe.RefundReasonRequestedByCustomer)),
	}
	switch {
	case payment.PaymentIntentID != "":
		params.PaymentIntent = stripe.String(payment.PaymentIntentID)
	case payment.ChargeID != "":
		params.Charge = stripe.String(payment.ChargeID)
	default:
		return nil, fmt.Errorf("payment has no refundable Stripe charge or payment intent")
	}

	refund, err := refundapi.New(params)
	if err != nil {
		return nil, err
	}

	chargeID := payment.ChargeID
	if refund.Charge != nil && refund.Charge.ID != "" {
		chargeID = refund.Charge.ID
	}
	if chargeID != "" {
		if charge, err := chargeapi.Get(chargeID, nil); err == nil {
			app.applyStripeChargeEvent(charge)
			return refund, nil
		}
	}

	app.setPayment(payment.ID, func(stored *Payment) {
		applyStripeRefundToPayment(refund, stored)
		stored.LastReconciledAt = time.Now()
	})
	return refund, nil
}

func (app *appContext) latestRefundablePayment(seed Payment) Payment {
	best := seed
	bestTime := paymentRefundSortTime(best)
	for _, payment := range app.storage.GetPayments() {
		if payment.Provider != lm.Stripe || payment.SubscriptionID == "" || payment.SubscriptionID != seed.SubscriptionID {
			continue
		}
		if payment.Amount-payment.RefundedAmount <= 0 {
			continue
		}
		if payment.PaymentIntentID == "" && payment.ChargeID == "" {
			continue
		}
		if best.PaymentIntentID == "" && best.ChargeID == "" {
			best = payment
			bestTime = paymentRefundSortTime(payment)
			continue
		}
		if t := paymentRefundSortTime(payment); t.After(bestTime) {
			best = payment
			bestTime = t
		}
	}
	return best
}

func paymentRefundSortTime(payment Payment) time.Time {
	if !payment.PaidAt.IsZero() {
		return payment.PaidAt
	}
	return payment.Created
}
