package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plutov/paypal/v4"
)

var (
	ppClient *paypal.Client
)

// InitPayPal initializes the PayPal client
func InitPayPal(config *Config) {
	clientID := config.Section("paypal").Key("client_id").String()
	secret := config.Section("paypal").Key("client_secret").String()
	mode := config.Section("paypal").Key("mode").String()

	var apiBase string
	if mode == "live" {
		apiBase = paypal.APIBaseLive
	} else {
		apiBase = paypal.APIBaseSandBox
	}

	var err error
	ppClient, err = paypal.NewClient(clientID, secret, apiBase)
	if err != nil {
		fmt.Printf("Error initializing PayPal client: %v\n", err)
		paypalEnabled = false
		return
	}
	_, err = ppClient.GetAccessToken(context.Background())
	if err != nil {
		fmt.Printf("Error getting PayPal access token: %v\n", err)
		paypalEnabled = false
		return
	}
	paypalEnabled = true
}

type createSubscriptionDTO struct {
	Email string `json:"email" binding:"required,email"`
}

// @Summary Create a PayPal Subscription (returns subscription ID and approve link)
// @Produce json
// @Param body body createSubscriptionDTO true "Subscription Request"
// @Success 200 {object} stringResponse
// @Router /paypal/create-subscription [post]
func (app *appContext) PostPayPalCreateSubscription(gc *gin.Context) {
	if !paypalEnabled {
		respond(400, "PayPal disabled", gc)
		return
	}

	var req createSubscriptionDTO
	if err := gc.ShouldBindJSON(&req); err != nil {
		respond(400, "Invalid request: "+err.Error(), gc)
		return
	}

	planID := app.config.Section("paypal").Key("plan_id_monthly").String()
	if planID == "" {
		respond(500, "Monthly Plan ID not configured", gc)
		return
	}

	// Double-Billing Prevention: Check if user exists and is active
	for _, em := range app.storage.GetEmails() {
		if strings.EqualFold(em.Addr, req.Email) {
			userID := em.JellyfinID
			// Check if user is effectively active (not disabled, expiry in future)
			user, err := app.jf.UserByID(userID, false)
			if err == nil && !user.Policy.IsDisabled {
				// Check expiry
				expiry := time.Now()
				if userExpiry, ok := app.storage.GetUserExpiryKey(userID); ok {
					expiry = userExpiry.Expiry
				}
				if expiry.After(time.Now()) {
					// User is active and not expired. strict prevention.
					app.info.Printf("Blocked duplicate subscription attempt for active user %s (%s)", userID, req.Email)
					respond(409, "You already have an active subscription.", gc)
					return
				}
			}
			break
		}
	}

	// Create Subscription
	sub := paypal.SubscriptionBase{
		PlanID: planID,
		Subscriber: &paypal.Subscriber{
			EmailAddress: req.Email,
		},
		ApplicationContext: &paypal.ApplicationContext{
			UserAction: "SUBSCRIBE_NOW",
			ReturnURL:  fmt.Sprintf("%s/payment/success", ExternalURI(gc)),
			CancelURL:  fmt.Sprintf("%s/store?canceled=true", ExternalURI(gc)),
		},
	}

	subscription, err := ppClient.CreateSubscription(context.Background(), sub)
	if err != nil {
		app.err.Printf("Failed to create PayPal subscription: %v", err)
		respond(500, "Failed to create subscription", gc)
		return
	}

	gc.JSON(200, gin.H{
		"subscriptionID": subscription.ID,
		"link":           subscription.Links[0].Href,
	})
}

type captureSubscriptionDTO struct {
	SubscriptionID string `json:"subscriptionID" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
}

// @Summary Capture/Verify PayPal Subscription
// @Produce json
// @Param body body captureSubscriptionDTO true "Capture Request"
// @Success 200 {object} stringResponse
// @Router /paypal/capture-subscription [post]
func (app *appContext) PostPayPalCaptureSubscription(gc *gin.Context) {
	if !paypalEnabled {
		respond(400, "PayPal disabled", gc)
		return
	}

	var req captureSubscriptionDTO
	if err := gc.ShouldBindJSON(&req); err != nil {
		respond(400, "Invalid request: "+err.Error(), gc)
		return
	}

	// Verify Subscription Status
	sub, err := ppClient.GetSubscriptionDetails(context.Background(), req.SubscriptionID)
	if err != nil {
		app.err.Printf("Failed to get PayPal subscription %s: %v", req.SubscriptionID, err)
		respond(500, "Failed to verify subscription", gc)
		return
	}

	// For Sandbox, status might be different, but ACTIVE is expected after approval
	app.info.Printf("PayPal Subscription %s status: %s", req.SubscriptionID, sub.SubscriptionStatus)

	if sub.SubscriptionStatus != "ACTIVE" && sub.SubscriptionStatus != "APPROVED" {
		// Sometimes it takes a moment, or it's APPROVED but not yet ACTIVE
		// We will proceed if APPROVED or ACTIVE.
		respond(400, fmt.Sprintf("Subscription not active (Status: %s)", sub.SubscriptionStatus), gc)
		return
	}

	// Check if user exists by email (Re-subscription logic)
	var existingUserID string
	var existingUserEmailStruct EmailAddress

	for _, em := range app.storage.GetEmails() {
		if strings.EqualFold(em.Addr, req.Email) {
			existingUserID = em.JellyfinID
			existingUserEmailStruct = em
			break
		}
	}

	if existingUserID != "" {
		// UPDATE EXISTING USER
		app.info.Printf("Existing user found for %s (%s). Reactivating subscription.", req.Email, existingUserID)

		// 0. Auto-Cancel Old Subscription to prevent Double Billing
		if strings.HasPrefix(existingUserEmailStruct.Label, "PayPal Subscription: ") {
			oldSubID := strings.TrimPrefix(existingUserEmailStruct.Label, "PayPal Subscription: ")
			if oldSubID != "" && oldSubID != req.SubscriptionID {
				app.info.Printf("Cancelling OLD Subscription %s for user %s to prevent double billing...", oldSubID, existingUserID)
				// Reason is required by PayPal API
				err := ppClient.CancelSubscription(context.Background(), oldSubID, "Re-subscribed via jfa-go")
				if err != nil {
					app.err.Printf("Failed to cancel old subscription %s: %v", oldSubID, err)
					// We continue anyway, as we don't want to block the new one.
				} else {
					app.info.Printf("Successfully cancelled old subscription %s", oldSubID)
				}
			}
		}

		// 1. Update Label to link NEW Subscription ID
		existingUserEmailStruct.Label = "PayPal Subscription: " + req.SubscriptionID
		app.storage.SetEmailsKey(existingUserID, existingUserEmailStruct)

		// 2. Extend Expiry
		expiry := time.Now()
		lastTx := ""
		if userExpiry, ok := app.storage.GetUserExpiryKey(existingUserID); ok {
			expiry = userExpiry.Expiry
			lastTx = userExpiry.LastTransactionID
		}

		// IDEMPOTENCY CHECK
		if lastTx == req.SubscriptionID {
			app.info.Printf("Transaction %s already processed for user %s. Skipping (Idempotent).", req.SubscriptionID, existingUserID)
			gc.JSON(200, gin.H{"success": true, "message": "Subscription already processed."})
			return
		}

		// If expired, start from now. If active, add to current time.
		if expiry.Before(time.Now()) {
			expiry = time.Now()
		}
		newExpiry := expiry.AddDate(0, 1, 0)
		app.storage.SetUserExpiryKey(existingUserID, UserExpiry{Expiry: newExpiry, LastTransactionID: req.SubscriptionID})

		// 3. Re-enable User if disabled (Force enable to ensure functionality)
		paramsUser, err := app.jf.UserByID(existingUserID, false)
		if err == nil {
			app.info.Printf("Ensuring user %s is enabled...", existingUserID)
			// Re-enable and Clear Cache
			err, _, _ = app.SetUserDisabled(paramsUser, false)
			if err != nil {
				app.err.Printf("Failed to re-enable user %s: %v", existingUserID, err)
			}
			app.InvalidateUserCaches()
		}

		app.info.Printf("SUCCESS: Reactivated user %s with new PayPal Subscription %s", existingUserID, req.SubscriptionID)

		// Return success (no invite needed)
		gc.JSON(200, gin.H{"success": true, "message": "Subscription reactivated for existing account."})
		return
	}

	// Generate Invite Code (New User Logic)
	inviteCode := GenerateInviteCode()
	profile := "Default" // Default profile
	if _, ok := app.storage.GetProfileKey(profile); !ok {
		profile = "Default"
	}

	invite := Invite{
		Code:          inviteCode,
		Created:       time.Now(),
		Label:         "PayPal Subscription " + req.SubscriptionID,
		UserLabel:     "PayPal Subscription: " + req.SubscriptionID,
		RemainingUses: 1,
		Profile:       profile,
		SendTo:        req.Email,
		ValidTill:     time.Now().AddDate(0, 1, 0), // 1 Month
		UserExpiry:    true,
		UserMonths:    1,
	}

	app.storage.SetInvitesKey(inviteCode, invite)
	app.info.Printf("SUCCESS: Generated Invite Code %s for %s via PayPal", inviteCode, req.Email)

	msg, err := app.email.constructInvite(&invite, false)
	if err != nil {
		app.err.Printf("Failed to construct invite email for %s: %v", req.Email, err)
	} else {
		err = app.email.send(msg, req.Email)
		if err != nil {
			app.err.Printf("Failed to send invite email to %s: %v", req.Email, err)
		} else {
			app.info.Printf("Sent purchased invite %s to %s", inviteCode, req.Email)
		}
	}

	gc.JSON(200, gin.H{"success": true, "inviteCode": inviteCode})
}

type paypalWebhookEvent struct {
	EventType string `json:"event_type"`
	Resource  struct {
		BillingAgreementID string `json:"billing_agreement_id"`
		ID                 string `json:"id"`
	} `json:"resource"`
}

// @Summary Handle PayPal Webhooks
// @Produce json
// @Router /paypal/webhook [post]
func (app *appContext) PostPayPalWebhook(gc *gin.Context) {
	// 1. Parse Event
	var event paypalWebhookEvent
	if err := gc.ShouldBindJSON(&event); err != nil {
		app.err.Printf("Error parsing PayPal webhook: %v", err)
		respond(400, "Invalid payload", gc)
		return
	}

	app.info.Printf("Received PayPal Webhook: %s", event.EventType)

	if event.EventType == "PAYMENT.SALE.COMPLETED" {
		subID := event.Resource.BillingAgreementID
		if subID == "" {
			app.err.Printf("PayPal Payment missing BillingAgreementID")
			return
		}

		app.info.Printf("Processing PayPal Payment for Subscription: %s", subID)

		// 2. Find User by Label
		var userID string
		var userEmail string
		targetLabel := "PayPal Subscription: " + subID

		for _, em := range app.storage.GetEmails() {
			if strings.Contains(em.Label, targetLabel) {
				userID = em.JellyfinID
				userEmail = em.Addr
				break
			}
		}

		if userID == "" {
			app.err.Printf("Could not find user for Subscription %s", subID)
			return
		}

		// 3. Extend Expiry
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

		// 4. Re-enable User if currently disabled (Force enable)
		paramsUser, err := app.jf.UserByID(userID, false)
		if err == nil {
			app.info.Printf("Ensuring user %s is enabled...", userID)
			err, _, _ = app.SetUserDisabled(paramsUser, false)
			if err != nil {
				app.err.Printf("Failed to re-enable user %s: %v", userID, err)
			}
			app.InvalidateUserCaches()
		}

		app.info.Printf("SUCCESS: Extended expiry for user %s (%s) to %s via PayPal Webhook", userID, userEmail, newExpiry)
	}

	gc.Status(200)
}
