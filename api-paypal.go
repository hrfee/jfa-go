package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
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
		log.Printf(lm.FailedInitPayPal, err)
		paypalEnabled = false
		return
	}
	_, err = ppClient.GetAccessToken(context.Background())
	if err != nil {
		log.Printf(lm.FailedInitPayPal, err)
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

	// Double-billing prevention
	if userID, _, found := app.findUserByEmail(req.Email); found {
		user, err := app.jf.UserByID(userID, false)
		if err == nil && !user.Policy.IsDisabled {
			expiry := time.Now()
			if userExpiry, ok := app.storage.GetUserExpiryKey(userID); ok {
				expiry = userExpiry.Expiry
			}
			if expiry.After(time.Now()) {
				app.info.Printf(lm.PayPalBlockedDuplicate, userID, req.Email)
				respond(409, "You already have an active subscription.", gc)
				return
			}
		}
	}

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
		app.err.Printf(lm.FailedCreatePayPalSubscription, err)
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

	sub, err := ppClient.GetSubscriptionDetails(context.Background(), req.SubscriptionID)
	if err != nil {
		app.err.Printf(lm.FailedGetPayPalSubscription, req.SubscriptionID, err)
		respond(500, "Failed to verify subscription", gc)
		return
	}

	app.debug.Printf("PayPal Subscription %s status: %s", req.SubscriptionID, sub.SubscriptionStatus)

	if sub.SubscriptionStatus != "ACTIVE" && sub.SubscriptionStatus != "APPROVED" {
		// Sometimes it takes a moment, or it's APPROVED but not yet ACTIVE
		// We will proceed if APPROVED or ACTIVE.
		respond(400, fmt.Sprintf("Subscription not active (Status: %s)", sub.SubscriptionStatus), gc)
		return
	}

	existingUserID, existingEmail, found := app.findUserByEmail(req.Email)
	if found {
		app.info.Printf(lm.ExistingUserFound, req.Email, existingUserID)

		// Auto-cancel old subscription to prevent double billing
		if strings.HasPrefix(existingEmail.Label, "PayPal Subscription: ") {
			oldSubID := strings.TrimPrefix(existingEmail.Label, "PayPal Subscription: ")
			if oldSubID != "" && oldSubID != req.SubscriptionID {
				err := ppClient.CancelSubscription(context.Background(), oldSubID, "Re-subscribed via jfa-go")
				if err != nil {
					app.err.Printf(lm.FailedCancelPayPalSubscription, oldSubID, err)
				} else {
					app.info.Printf(lm.PayPalCancelledOldSubscription, oldSubID, existingUserID)
				}
			}
		}

		existingEmail.Label = "PayPal Subscription: " + req.SubscriptionID
		app.storage.SetEmailsKey(existingUserID, existingEmail)

		expiry := time.Now()
		lastTx := ""
		if userExpiry, ok := app.storage.GetUserExpiryKey(existingUserID); ok {
			expiry = userExpiry.Expiry
			lastTx = userExpiry.LastTransactionID
		}

		if lastTx == req.SubscriptionID {
			app.info.Printf(lm.PayPalTransactionAlreadyProcessed, req.SubscriptionID, existingUserID)
			gc.JSON(200, gin.H{"success": true, "message": "Subscription already processed."})
			return
		}

		if expiry.Before(time.Now()) {
			expiry = time.Now()
		}
		newExpiry := expiry.AddDate(0, 1, 0)
		app.storage.SetUserExpiryKey(existingUserID, UserExpiry{Expiry: newExpiry, LastTransactionID: req.SubscriptionID})

		if paramsUser, err := app.jf.UserByID(existingUserID, false); err == nil {
			if err, _, _ = app.SetUserDisabled(paramsUser, false); err != nil {
				app.err.Printf(lm.FailedReEnableUser, existingUserID, err)
			}
			app.InvalidateUserCaches()
		}

		app.info.Printf(lm.UserReactivated, existingUserID, newExpiry)
		gc.JSON(200, gin.H{"success": true, "message": "Subscription reactivated for existing account."})
		return
	}

	// New user: generate invite
	inviteCode := GenerateInviteCode()
	profile := "Default"
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
		ValidTill:     time.Now().AddDate(0, 1, 0),
		UserExpiry:    true,
		UserMonths:    1,
	}

	app.storage.SetInvitesKey(inviteCode, invite)
	app.info.Printf(lm.GeneratedInviteForPurchase, inviteCode, req.Email)

	if emailEnabled {
		msg, err := app.email.constructInvite(&invite, false)
		if err != nil {
			app.err.Printf(lm.FailedConstructInviteMessage, req.Email, err)
		} else if err = app.email.send(msg, req.Email); err != nil {
			app.err.Printf(lm.FailedSendInviteMessage, inviteCode, req.Email, err)
		} else {
			app.info.Printf(lm.SentInviteMessage, inviteCode, req.Email)
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
	var event paypalWebhookEvent
	if err := gc.ShouldBindJSON(&event); err != nil {
		app.err.Printf(lm.PayPalWebhookReceived, "parse error: "+err.Error())
		respond(400, "Invalid payload", gc)
		return
	}

	app.info.Printf(lm.PayPalWebhookReceived, event.EventType)

	switch event.EventType {
	case "PAYMENT.SALE.COMPLETED":
		app.handlePayPalPayment(event)
	case "BILLING.SUBSCRIPTION.CANCELLED":
		app.handlePayPalCancellation(event)
	}

	gc.Status(200)
}

func (app *appContext) handlePayPalPayment(event paypalWebhookEvent) {
	subID := event.Resource.BillingAgreementID
	if subID == "" {
		app.err.Printf(lm.PayPalPaymentReceived, "(missing BillingAgreementID)")
		return
	}

	app.info.Printf(lm.PayPalPaymentReceived, subID)

	// Find user by subscription label
	var userID, userEmail string
	targetLabel := "PayPal Subscription: " + subID
	for _, em := range app.storage.GetEmails() {
		if strings.Contains(em.Label, targetLabel) {
			userID = em.JellyfinID
			userEmail = em.Addr
			break
		}
	}

	if userID == "" {
		app.err.Printf(lm.FailedFindUserByEmail, "subscription "+subID)
		return
	}

	expiry := time.Now()
	if userExpiry, ok := app.storage.GetUserExpiryKey(userID); ok {
		expiry = userExpiry.Expiry
	}
	if expiry.Before(time.Now()) {
		expiry = time.Now()
	}
	newExpiry := expiry.AddDate(0, 1, 0)

	app.storage.SetUserExpiryKey(userID, UserExpiry{Expiry: newExpiry})

	if paramsUser, err := app.jf.UserByID(userID, false); err == nil {
		if err, _, _ = app.SetUserDisabled(paramsUser, false); err != nil {
			app.err.Printf(lm.FailedReEnableUser, userID, err)
		}
		app.InvalidateUserCaches()
	}

	app.info.Printf(lm.UserExpiryExtended, userID, userEmail, newExpiry)
}

func (app *appContext) handlePayPalCancellation(event paypalWebhookEvent) {
	subID := event.Resource.ID
	if subID == "" {
		app.err.Printf(lm.PayPalSubscriptionCancelled, "(missing ID)")
		return
	}

	app.info.Printf(lm.PayPalSubscriptionCancelled, subID)

	var userID string
	targetLabel := "PayPal Subscription: " + subID
	for _, em := range app.storage.GetEmails() {
		if strings.Contains(em.Label, targetLabel) {
			userID = em.JellyfinID
			break
		}
	}

	if userID == "" {
		app.err.Printf(lm.FailedFindUserByEmail, "subscription "+subID)
		return
	}

	app.storage.SetUserExpiryKey(userID, UserExpiry{Expiry: time.Now().Add(-1 * time.Second)})

	if paramsUser, err := app.jf.UserByID(userID, false); err == nil {
		if err, _, _ = app.SetUserDisabled(paramsUser, true); err != nil {
			app.err.Printf(lm.FailedDisableUser, userID, err)
		} else {
			app.info.Printf(lm.UserDisabledDueToCancellation, userID)
		}
		app.InvalidateUserCaches()
	}
}
