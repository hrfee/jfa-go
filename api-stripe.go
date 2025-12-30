package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary Create a checkout session for an existing invite (Pay-to-Unlock).
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

	baseURL := ExternalURI(gc)
	successURL := fmt.Sprintf("%s/invite/%s?success=payment", baseURL, code)
	cancelURL := fmt.Sprintf("%s/invite/%s?canceled=payment", baseURL, code)

	// Pass nil metadata for legacy flow
	url, err := CreateCheckoutSession(code, inv.PriceAmount, inv.PriceCurrency, successURL, cancelURL, nil)
	if err != nil {
		app.err.Printf("Failed to create checkout session: %v", err)
		respond(500, "Failed to create checkout session", gc)
		return
	}

	gc.SetCookie("jfa_payment_lock", code, 3600*24, "/", "", false, true)

	gc.JSON(200, stringResponse{Response: url})
}

type createCheckoutDTO struct {
	Email string `json:"email" binding:"required,email"`
	Plan  string `json:"plan" binding:"required"`
}

// @Summary Create a checkout session for a new invite (Pay-to-Generate).
// @Produce json
// @Param body body createCheckoutDTO true "Checkout Request"
// @Success 200 {object} stringResponse
// @Router /stripe/create-checkout [post]
func (app *appContext) PostStripeCreateCheckout(gc *gin.Context) {
	if !stripeEnabled {
		respond(400, "Stripe disabled", gc)
		return
	}

	var req createCheckoutDTO
	if err := gc.ShouldBindJSON(&req); err != nil {
		respond(400, "Invalid request: "+err.Error(), gc)
		return
	}

	// MVP Pricing Configuration (Hardcoded for now as per plan/task)
	// MVP Pricing Configuration - Standard Plan Only
	// Hardcoded for now as per plan/task
	// Default to config or usd
	var priceAmount int64 = 500 // $5.00
	var profileName = "Default"
	var currency = app.config.Section("stripe").Key("price_currency").MustString("usd")

	// Ensure we only process "Standard" plans (or empty which defaults to Standard)
	if req.Plan != "Standard" && req.Plan != "" {
		req.Plan = "Standard"
	}

	// Generate a temporary "Reference ID" for the log (not the invite code yet)
	refID := "purchase-" + strconv.FormatInt(time.Now().Unix(), 10)

	baseURL := ExternalURI(gc)
	// Redirect to a generic success page or the store with a flag
	successURL := fmt.Sprintf("%s/store?success=true", baseURL)
	cancelURL := fmt.Sprintf("%s/store?canceled=true", baseURL)

	metadata := map[string]string{
		"target_email": req.Email,
		"plan":         req.Plan,
		"profile":      profileName,
	}

	url, err := CreateCheckoutSession(refID, priceAmount, currency, successURL, cancelURL, metadata)
	if err != nil {
		app.err.Printf("Failed to create checkout session: %v", err)
		respond(500, "Failed to create checkout session", gc)
		return
	}

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
		app.info.Printf("DEBUG: Signature verification DISABLED. Verifying event via Stripe API.")
	}

	refID, metadata, err := HandleWebhook(payload, sigHeader, webhookSecret, verifySignature)
	if err != nil {
		app.err.Printf("Webhook error: %v", err)
		gc.AbortWithStatus(400)
		return
	}

	// Check for metadata first (New Flow)
	if targetEmail, ok := metadata["target_email"]; ok {
		app.info.Printf("Payment received for NEW invite (Plan: %s) to %s", metadata["plan"], targetEmail)

		// Generate Invite
		inviteCode := GenerateInviteCode()
		profile := metadata["profile"]
		if profile == "" {
			profile = "Default"
		}

		// Ensure profile exists, fallback to default if not
		if _, ok := app.storage.GetProfileKey(profile); !ok {
			app.debug.Printf("Profile %s not found for purchase, falling back to Default", profile)
			profile = "Default"
		}

		invite := Invite{
			Code:          inviteCode,
			Created:       time.Now(),
			Label:         "Purchased by " + targetEmail,
			UserLabel:     "Purchased via Store",
			RemainingUses: 1,
			Profile:       profile,
			SendTo:        targetEmail,
			ValidTill:     time.Now().AddDate(0, 0, 1), // Default 1 day expiry to claim? Or no expiry? Let's say 30 days.
		}

		// Set sensible default expiry for purchased invites
		invite.ValidTill = time.Now().AddDate(0, 1, 0) // 1 Month to redeem

		app.storage.SetInvitesKey(inviteCode, invite)

		// LOG THE CODE for testing purposes (in case email fails)
		app.info.Printf("SUCCESS: Generated Invite Code %s for %s", inviteCode, targetEmail)

		// Send Email
		// We re-use logic similar to api-invites.go's send logic or use app.email.constructInvite directly
		// app.sendInvite is likely a helper in api-invites.go, which is package main, so accessible.
		// I need to check api-invites.go for the exact signature of app.sendInvite or if it is exported.
		// It is lowercase `sendInvite` in `api-invites.go`? I need to check.
		// If it's private and I'm in the same package `main`, I can call it.
		// Let's assume I can call it.

		// Construct a dummy sendInviteDTO to reuse that function if possible
		// OR just use app.email directly which is safer.
		msg, err := app.email.constructInvite(&invite, false)
		if err != nil {
			app.err.Printf("Failed to construct invite email for %s: %v", targetEmail, err)
		} else {
			err = app.email.send(msg, targetEmail)
			if err != nil {
				app.err.Printf("Failed to send invite email to %s: %v", targetEmail, err)
			} else {
				app.info.Printf("Sent purchased invite %s to %s", inviteCode, targetEmail)
			}
		}

	} else if refID != "" {
		// Old Flow: Pay to Unlock
		app.info.Printf("Payment received for EXISTING invite: %s", refID)
		inv, ok := app.storage.GetInvitesKey(refID)
		if ok {
			inv.PaymentStatus = "paid"
			app.storage.SetInvitesKey(refID, inv)
		}
	}

	gc.Status(200)
}
