package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v86"
	subscriptionapi "github.com/stripe/stripe-go/v86/subscription"
)

// @Summary Cancel the logged-in user's Stripe subscription at period end.
// @Produce json
// @Success 200 {object} MySubscriptionDTO
// @Failure 400 {object} stringResponse
// @Failure 404 {object} stringResponse
// @Failure 500 {object} stringResponse
// @Router /my/subscription/cancel [post]
// @Security Bearer
// @tags User Page
func (app *appContext) CancelMySubscription(gc *gin.Context) {
	if !stripeEnabled {
		respond(400, "Stripe disabled", gc)
		return
	}

	userID := gc.GetString("jfId")
	payment, ok := app.stripeSubscriptionForUser(userID)
	if !ok || payment.SubscriptionID == "" {
		respond(404, "No Stripe subscription found", gc)
		return
	}
	if payment.SubscriptionStatus == string(stripe.SubscriptionStatusCanceled) || payment.Status == paymentStatusSubscriptionCanceled {
		respond(400, "Subscription is already canceled", gc)
		return
	}

	sub, err := subscriptionapi.Update(payment.SubscriptionID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	})
	if err != nil {
		app.err.Printf("Failed to schedule Stripe subscription %s cancellation for user %s: %v", payment.SubscriptionID, userID, err)
		respond(500, "Failed to cancel subscription", gc)
		return
	}

	app.applyStripeSubscriptionEvent(sub)
	app.notifyStripeSubscriptionCancellation(sub, "user")
	app.info.Printf("Stripe subscription %s scheduled for cancellation by user %s", sub.ID, userID)

	if dto := app.mySubscriptionDTO(userID); dto != nil {
		gc.JSON(200, dto)
		return
	}
	gc.JSON(200, MySubscriptionDTO{
		Provider:          payment.Provider,
		SubscriptionID:    sub.ID,
		Status:            string(sub.Status),
		CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
		CancelAt:          sub.CancelAt,
		CanceledAt:        sub.CanceledAt,
		EndedAt:           sub.EndedAt,
	})
}
