package main

import (
	"fmt"
	"io"
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"
)

// @Summary Create a checkout session for an invite.
// @Produce json
// @Param code path string true "Invite Code"
// @Success 200 {object} stringResponse
// @Failure 400 {object} stringResponse
// @Router /stripe/checkout/{code} [post]
func (app *appContext) PostStripeCheckout(gc *gin.Context) {
	if !stripeEnabled {
		respond(400, "Stripe disabled", gc)
		return
	}
	code := gc.Param("code")
	inv, ok := app.storage.GetInvitesKey(code)
	if !ok {
		respond(400, "Invalid invite code", gc)
		return
	}

	if !inv.RequiredPayment || inv.PriceAmount == 0 {
		respond(200, "Payment not required", gc)
		return
	}

	// Construct success/cancel URLs
	// Assuming logic: Success -> returns to form with a flag? or a dedicated success page?
	// For now, let's redirect back to the form with ?paid=true
	baseURL := ExternalURI(gc)
	// Clean up baseURL to ensure it doesn't duplicate parts if auto-detected

	successURL := fmt.Sprintf("%s/invite/%s?success=payment", baseURL, code)
	cancelURL := fmt.Sprintf("%s/invite/%s?canceled=payment", baseURL, code)

	url, err := CreateCheckoutSession(code, inv.PriceAmount, inv.PriceCurrency, successURL, cancelURL)
	if err != nil {
		app.err.Printf("Failed to create checkout session: %v", err)
		respond(500, "Failed to create checkout session", gc)
		return
	}

	// Set a cookie to lock this payment session to this browser.
	// This prevents users from sharing a paid invite link.
	// MaxAge: 24 hours. Path: /. HttpOnly: true. Secure: false (for now, ideally true in prod).
	gc.SetCookie("jfa_payment_lock", code, 3600*24, "/", "", false, true)

	gc.JSON(200, stringResponse{Response: url})
}

// @Summary Handle Stripe Webhooks
// @Router /stripe/webhook [post]
func (app *appContext) StripeWebhook(gc *gin.Context) {
	if !stripeEnabled {
		gc.AbortWithStatus(404)
		return
	}

	const MaxBodyBytes = int64(65536)
	gc.Request.Body = http.MaxBytesReader(gc.Writer, gc.Request.Body, MaxBodyBytes)
	payload, err := io.ReadAll(gc.Request.Body)
	if err != nil {
		app.err.Printf("Error reading request body: %v", err)
		gc.AbortWithStatus(400)
		return
	}

	sigHeader := gc.GetHeader("Stripe-Signature")
	webhookSecret := strings.TrimSpace(app.config.Section("stripe").Key("webhook_secret").String())
	verifySignature := app.config.Section("stripe").Key("verify_signature").MustBool(false)

	if !verifySignature {
		app.info.Printf("DEBUG: Signature verification DISABLED. Verifying event via Stripe API (Two-Way Confirm).")
	}

	inviteCode, err := HandleWebhook(payload, sigHeader, webhookSecret, verifySignature)
	if err != nil {
		app.err.Printf("Webhook error: %v. Payload len: %d, Sig len: %d, Secret: %s...", err, len(payload), len(sigHeader), webhookSecret[:10])
		gc.AbortWithStatus(400)
		return
	}

	if inviteCode != "" {
		app.info.Printf("Payment received for invite: %s", inviteCode)
		// Mark invite as paid
		inv, ok := app.storage.GetInvitesKey(inviteCode)
		if ok {
			inv.PaymentStatus = "paid"
			app.storage.SetInvitesKey(inviteCode, inv)

			// Potential TODO: Trigger user creation if it was pending?
			// For now, we just mark it as paid. The user form will handle the rest
			// (user submits form -> we check if paid -> create user).
		}
	}

	gc.Status(200)
}
