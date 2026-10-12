package main

import (
	"strings"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v86"
)

var stripeSubscriptionCancellationNotifyMu sync.Mutex

func (app *appContext) notifyStripeSubscriptionCancellation(sub *stripe.Subscription, source string) {
	if sub == nil || sub.ID == "" || !emailEnabled || app.email == nil || app.email.sender == nil {
		return
	}

	targetEmail := stripeSubscriptionTargetEmail(sub, app)
	if targetEmail == "" {
		return
	}
	if !app.reserveStripeSubscriptionCancellationNotification(sub.ID) {
		return
	}

	userID, _, found := app.findUserByEmail(targetEmail)
	username := targetEmail
	if found {
		if user, err := app.jf.UserByID(userID, false); err == nil && user.Name != "" {
			username = user.Name
		}
	}

	msg := app.constructStripeSubscriptionCancellationMessage(username, userID, sub, source)
	if err := app.email.send(msg, targetEmail); err != nil {
		app.clearStripeSubscriptionCancellationNotified(sub.ID)
		app.err.Printf("Failed to send Stripe subscription cancellation email for %s to %s: %v", sub.ID, targetEmail, err)
		return
	}

	app.info.Printf("Sent Stripe subscription cancellation email for %s to %s", sub.ID, targetEmail)
}

func (app *appContext) constructStripeSubscriptionCancellationMessage(username, userID string, sub *stripe.Subscription, source string) *Message {
	serverName := serverHeader(app.config, nil)
	lines := []string{
		"Hi " + username + ",",
		"",
		"Your " + serverName + " subscription has been canceled.",
	}
	if source == "user" {
		lines = append(lines, "This change was requested from your account page.")
	} else if source == "admin" {
		lines = append(lines, "This change was made by an administrator.")
	} else {
		lines = append(lines, "This change was received from Stripe.")
	}

	cancelAt := app.stripeSubscriptionCancellationTime(userID, sub)
	if !cancelAt.IsZero() && cancelAt.After(time.Now()) {
		lines = append(lines, "Your account will remain available until "+formatDatetime(cancelAt)+".")
	} else {
		lines = append(lines, "Your account access may be disabled soon.")
	}

	lines = append(lines, "", "If you did not request this, contact the administrator.")
	return &Message{
		Subject: "Subscription cancellation update - " + serverName,
		Text:    strings.Join(lines, "\n"),
	}
}

func (app *appContext) stripeSubscriptionCancellationTime(userID string, sub *stripe.Subscription) time.Time {
	if sub != nil {
		if sub.CancelAt > 0 {
			return time.Unix(sub.CancelAt, 0)
		}
		if sub.EndedAt > 0 {
			return time.Unix(sub.EndedAt, 0)
		}
		if sub.CanceledAt > 0 && !sub.CancelAtPeriodEnd {
			return time.Unix(sub.CanceledAt, 0)
		}
	}
	if userID != "" {
		if expiry, ok := app.storage.GetUserExpiryKey(userID); ok {
			return expiry.Expiry
		}
	}
	return time.Time{}
}

func (app *appContext) stripeSubscriptionCancellationNotified(subscriptionID string) bool {
	for _, payment := range app.storage.GetPayments() {
		if payment.SubscriptionID == subscriptionID && !payment.SubscriptionCancelNotifiedAt.IsZero() {
			return true
		}
	}
	return false
}

func (app *appContext) reserveStripeSubscriptionCancellationNotification(subscriptionID string) bool {
	stripeSubscriptionCancellationNotifyMu.Lock()
	defer stripeSubscriptionCancellationNotifyMu.Unlock()

	if app.stripeSubscriptionCancellationNotified(subscriptionID) {
		return false
	}
	now := time.Now()
	marked := false
	app.setPaymentsByStripeIDs("", "", "", subscriptionID, func(payment *Payment) {
		if payment.SubscriptionCancelNotifiedAt.IsZero() {
			payment.SubscriptionCancelNotifiedAt = now
			marked = true
		}
	})
	return marked
}

func (app *appContext) clearStripeSubscriptionCancellationNotified(subscriptionID string) {
	stripeSubscriptionCancellationNotifyMu.Lock()
	defer stripeSubscriptionCancellationNotifyMu.Unlock()

	app.setPaymentsByStripeIDs("", "", "", subscriptionID, func(payment *Payment) {
		payment.SubscriptionCancelNotifiedAt = time.Time{}
	})
}
