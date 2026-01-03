package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v79"
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
	url, err := CreateCheckoutSession(code, inv.PriceAmount, inv.PriceCurrency, successURL, cancelURL, nil, "")
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

	// Pricing Logic
	// Pricing Logic
	var priceAmount int64
	var interval string
	var profileName = "Default"

	// Double-Billing Prevention: Check if user exists and is active
	// Only for Monthly (Subscription) plans
	if req.Plan == "Monthly" {
		for _, em := range app.storage.GetEmails() {
			if strings.EqualFold(em.Addr, req.Email) {
				userID := em.JellyfinID
				// Check if user is effectively active (not disabled, expiry in future)
				// Check if user is effectively active (not disabled, expiry in future)
				user, err := app.jf.UserByID(userID, false)
				if err == nil && !user.Policy.IsDisabled {
					// Check expiry
					expiry := time.Now()
					if userExpiry, ok := app.storage.GetUserExpiryKey(userID); ok {
						expiry = userExpiry.Expiry
					}

					// This block seems to be misplaced. `invoice.ID` is not available here.
					// Assuming the intent was to check for an *active* subscription and prevent a new one.
					// The original logic for checking active expiry is more appropriate here.
					if expiry.After(time.Now()) {
						app.info.Printf("Blocked duplicate Stripe subscription attempt for active user %s (%s)", userID, req.Email)
						respond(409, "You already have an active subscription.", gc)
						return
					}
				}
				break
			}
		}
	}
	var currency = app.config.Section("stripe").Key("price_currency").MustString("usd")
	priceStandard := app.config.Section("stripe").Key("price_standard").MustInt64(500)
	priceMonthly := app.config.Section("stripe").Key("price_monthly").MustInt64(200)

	if req.Plan == "Monthly" {
		priceAmount = priceMonthly
		interval = "month"
	} else {
		// Default to Standard
		req.Plan = "Standard"
		priceAmount = priceStandard // $5.00
		interval = ""               // One-time
	}

	// Generate a temporary "Reference ID" for the log (not the invite code yet)
	refID := "purchase-" + strconv.FormatInt(time.Now().Unix(), 10)

	baseURL := ExternalURI(gc)
	// Redirect to a generic success page or the store with a flag
	successURL := fmt.Sprintf("%s/payment/success", baseURL)
	cancelURL := fmt.Sprintf("%s/store?canceled=true", baseURL)

	metadata := map[string]string{
		"target_email": req.Email,
		"plan":         req.Plan,
		"profile":      profileName,
	}

	url, err := CreateCheckoutSession(refID, priceAmount, currency, successURL, cancelURL, metadata, interval)
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

	event, err := HandleWebhook(payload, sigHeader, webhookSecret, verifySignature)
	if err != nil {
		app.err.Printf("Webhook error: %v", err)
		gc.AbortWithStatus(400)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			app.err.Printf("Error parsing webhook JSON: %v", err)
			return
		}

		refID := session.ClientReferenceID
		metadata := session.Metadata

		// Check for metadata first (New Flow)
		if targetEmail, ok := metadata["target_email"]; ok {
			app.info.Printf("Payment received for Checkout Session (Plan: %s) to %s", metadata["plan"], targetEmail)

			// Check if user exists by email (Re-subscription logic)
			var existingUserID string
			var existingUserEmailStruct EmailAddress

			for _, em := range app.storage.GetEmails() {
				if strings.EqualFold(em.Addr, targetEmail) {
					existingUserID = em.JellyfinID
					existingUserEmailStruct = em
					break
				}
			}

			if existingUserID != "" {
				// UPDATE EXISTING USER (Re-activation)
				app.info.Printf("Existing user found for %s (%s). Reactivating subscription via Store Checkout.", targetEmail, existingUserID)

				// 1. Update Label to link Stripe ID (if needed, though Subscription ID isn't in Session object easily unless we expand)
				// For now, we keep "Purchased via Store" or similar, or try to get Sub ID from invoice.
				// Simply updating label to "Stripe Subscription: Active" is enough for our checks.
				if subscriptionID := session.Subscription; subscriptionID != nil {
					existingUserEmailStruct.Label = "Stripe Subscription: " + subscriptionID.ID
					app.storage.SetEmailsKey(existingUserID, existingUserEmailStruct)
				} else {
					// One-time payment or ID missing
					existingUserEmailStruct.Label = "Purchased via Store"
					app.storage.SetEmailsKey(existingUserID, existingUserEmailStruct)
				}

				// 2. Extend Expiry
				expiry := time.Now()
				lastTx := ""
				if userExpiry, ok := app.storage.GetUserExpiryKey(existingUserID); ok {
					expiry = userExpiry.Expiry
					lastTx = userExpiry.LastTransactionID
				}

				// IDEMPOTENCY CHECK
				if lastTx == session.ID {
					app.info.Printf("Stripe Session %s already processed for user %s. Skipping.", session.ID, existingUserID)
					return // Success
				}

				if expiry.Before(time.Now()) {
					expiry = time.Now()
				}
				var newExpiry time.Time
				if metadata["plan"] == "Monthly" {
					newExpiry = expiry.AddDate(0, 1, 0)
				} else {
					newExpiry = expiry.AddDate(10, 0, 0) // Lifetime
				}
				app.storage.SetUserExpiryKey(existingUserID, UserExpiry{Expiry: newExpiry, LastTransactionID: session.ID})

				// 3. Re-enable User if currently disabled (Force enable)
				paramsUser, err := app.jf.UserByID(existingUserID, false)
				if err == nil {
					app.info.Printf("Ensuring user %s is enabled...", existingUserID)
					err, _, _ = app.SetUserDisabled(paramsUser, false)
					if err != nil {
						app.err.Printf("Failed to re-enable user %s: %v", existingUserID, err)
					}
					app.InvalidateUserCaches()
				}

				app.info.Printf("SUCCESS: Reactivated user %s to %s via Stripe Checkout", existingUserID, newExpiry)

				// Don't generate invite code.
			} else {
				// NEW USER LOGIC
				inviteCode := GenerateInviteCode()
				profile := metadata["profile"]
				if profile == "" {
					profile = "Default"
				}
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
				}

				if metadata["plan"] == "Monthly" {
					invite.ValidTill = time.Now().AddDate(0, 1, 0) // 1 Month to redeem
					invite.UserExpiry = true
					invite.UserMonths = 1
				} else {
					invite.ValidTill = time.Now().AddDate(10, 0, 0)
					invite.UserExpiry = false
				}

				app.storage.SetInvitesKey(inviteCode, invite)
				app.info.Printf("SUCCESS: Generated Invite Code %s for %s", inviteCode, targetEmail)

				// Email Sending with Safety Checks + Async
				if app.config.Section("email").Key("enabled").MustBool(false) {
					go func(inv Invite, tEmail string) {
						msg, err := app.email.constructInvite(&inv, false)
						if err != nil {
							app.err.Printf("Failed to construct invite email for %s: %v", tEmail, err)
						} else {
							err = app.email.send(msg, tEmail)
							if err != nil {
								app.err.Printf("Failed to send invite email to %s: %v", tEmail, err)
							} else {
								app.info.Printf("Sent purchased invite %s to %s", inv.Code, tEmail)
							}
						}
					}(invite, targetEmail)
				} else {
					app.info.Printf("Email disabled in config. Skipping invite email for %s", targetEmail)
				}
			}
			// [Old Flow Logic: Pay to Unlock]
			app.info.Printf("Payment received for EXISTING invite: %s", refID)
			inv, ok := app.storage.GetInvitesKey(refID)
			if ok {
				inv.PaymentStatus = "paid"
				app.storage.SetInvitesKey(refID, inv)
			}
		}

	case "invoice.payment_succeeded":
		var invoice stripe.Invoice
		err := json.Unmarshal(event.Data.Raw, &invoice)
		if err != nil {
			app.err.Printf("Error parsing webhook JSON: %v", err)
			return
		}

		// Ensure this is a subscription renewal
		if invoice.BillingReason == stripe.InvoiceBillingReasonSubscriptionCycle {
			if invoice.CustomerEmail == "" {
				app.err.Printf("Invoice %s has no customer email, cannot extend user expiry", invoice.ID)
				return
			}
			email := invoice.CustomerEmail
			app.info.Printf("Subscription RENEWAL received for %s", email)

			// Find Jellyfin User by Email
			// We iterate because we don't have a direct Email -> UserID index easily exposed here
			// But app.storage.GetEmailsKey(userID) gives email...
			// Wait, app.EmailAddressExists(addr) exists?
			// app.storage.GetEmails() returns all.

			// Ideally we'd have a helper, but let's just loop over emails for now as it's MVP
			var userID string
			for _, em := range app.storage.GetEmails() {
				if strings.EqualFold(em.Addr, email) {
					userID = em.JellyfinID
					break
				}
			}

			if userID == "" {
				app.err.Printf("Could not find user with email %s to extend expiry", email)
				return
			}

			// Extend User Expiry by 1 Month
			expiry := time.Now()
			userExpiry, ok := app.storage.GetUserExpiryKey(userID)
			if ok {
				expiry = userExpiry.Expiry
			}
			// If current expiry is in the past, start from NOW. If in future, add to it.
			if expiry.Before(time.Now()) {
				expiry = time.Now()
			}
			newExpiry := expiry.AddDate(0, 1, 0) // Add 1 Month

			app.storage.SetUserExpiryKey(userID, UserExpiry{Expiry: newExpiry})

			// Hardening: Re-enable User if currently disabled (Force enable)
			paramsUser, err := app.jf.UserByID(userID, false)
			if err == nil {
				app.info.Printf("Ensuring user %s is enabled...", userID)
				err, _, _ = app.SetUserDisabled(paramsUser, false)
				if err != nil {
					app.err.Printf("Failed to re-enable user %s: %v", userID, err)
				}
				app.InvalidateUserCaches()
			}

			app.info.Printf("SUCCESS: Extended expiry for user %s (%s) to %s", userID, email, newExpiry)
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &sub)
		if err != nil {
			app.err.Printf("Error parsing webhook JSON: %v", err)
			return
		}

		// Attempt to identify user by Metadata (preferred) or lookup (if we stored sub ID)
		targetEmail := sub.Metadata["target_email"]
		if targetEmail == "" {
			app.info.Printf("Webhook: customer.subscription.deleted received for %s but no target_email in metadata. Ignoring active revocation (Passive Expiry will still apply).", sub.ID)
			return
		}

		app.info.Printf("Active Revocation: Subscription %s deleted for %s. Disabling user...", sub.ID, targetEmail)

		// 1. Find User
		var userID string
		for _, em := range app.storage.GetEmails() {
			if strings.EqualFold(em.Addr, targetEmail) {
				userID = em.JellyfinID
				break
			}
		}

		if userID == "" {
			app.err.Printf("Could not find user with email %s to revoke access", targetEmail)
			return
		}

		// 2. Expire immediately (Set expiry to 1 second ago)
		app.storage.SetUserExpiryKey(userID, UserExpiry{Expiry: time.Now().Add(-1 * time.Second)})

		// 3. Disable User (Hard Revoke)
		paramsUser, err := app.jf.UserByID(userID, false)
		if err == nil {
			err, _, _ = app.SetUserDisabled(paramsUser, true)
			if err != nil {
				app.err.Printf("Failed to disable user %s: %v", userID, err)
			} else {
				app.info.Printf("SUCCESS: Disabled user %s due to subscription cancellation", userID)
			}
			app.InvalidateUserCaches()
		}
	}

	gc.Status(200)
}
