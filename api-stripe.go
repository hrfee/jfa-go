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
	lm "github.com/hrfee/jfa-go/logmessages"
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

	url, err := CreateCheckoutSession(code, inv.PriceAmount, inv.PriceCurrency, successURL, cancelURL, nil, "")
	if err != nil {
		app.err.Printf(lm.FailedCreateCheckoutSession, err)
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

	var priceAmount int64
	var interval string
	var profileName = "Default"

	// Double-billing prevention for subscription plans
	if req.Plan == "Monthly" {
		if userID, _, found := app.findUserByEmail(req.Email); found {
			user, err := app.jf.UserByID(userID, false)
			if err == nil && !user.Policy.IsDisabled {
				expiry := time.Now()
				if userExpiry, ok := app.storage.GetUserExpiryKey(userID); ok {
					expiry = userExpiry.Expiry
				}
				if expiry.After(time.Now()) {
					app.info.Printf(lm.StripeBlockedDuplicate, userID, req.Email)
					respond(409, "You already have an active subscription.", gc)
					return
				}
			}
		}
	}

	currency := app.config.Section("stripe").Key("price_currency").MustString("usd")
	priceStandard := app.config.Section("stripe").Key("price_standard").MustInt64(500)
	priceMonthly := app.config.Section("stripe").Key("price_monthly").MustInt64(200)

	if req.Plan == "Monthly" {
		priceAmount = priceMonthly
		interval = "month"
	} else {
		req.Plan = "Standard"
		priceAmount = priceStandard
		interval = ""
	}

	refID := "purchase-" + strconv.FormatInt(time.Now().Unix(), 10)

	baseURL := ExternalURI(gc)
	successURL := fmt.Sprintf("%s/payment/success", baseURL)
	cancelURL := fmt.Sprintf("%s/store?canceled=true", baseURL)

	metadata := map[string]string{
		"target_email": req.Email,
		"plan":         req.Plan,
		"profile":      profileName,
	}

	url, err := CreateCheckoutSession(refID, priceAmount, currency, successURL, cancelURL, metadata, interval)
	if err != nil {
		app.err.Printf(lm.FailedCreateCheckoutSession, err)
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
		app.err.Printf(lm.FailedReading, "request body", err)
		gc.AbortWithStatus(400)
		return
	}

	sigHeader := gc.GetHeader("Stripe-Signature")
	webhookSecret := strings.TrimSpace(app.config.Section("stripe").Key("webhook_secret").String())
	verifySignature := app.config.Section("stripe").Key("verify_signature").MustBool(false)

	if !verifySignature {
		app.debug.Println(lm.StripeSignatureBypass)
	}

	event, err := HandleWebhook(payload, sigHeader, webhookSecret, verifySignature)
	if err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		gc.AbortWithStatus(400)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		app.handleStripeCheckoutCompleted(event)
	case "invoice.payment_succeeded":
		app.handleStripeInvoiceSucceeded(event)
	case "customer.subscription.deleted":
		app.handleStripeSubscriptionDeleted(event)
	}

	gc.Status(200)
}

func (app *appContext) handleStripeCheckoutCompleted(event *stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	refID := session.ClientReferenceID
	metadata := session.Metadata

	targetEmail, ok := metadata["target_email"]
	if !ok {
		// Legacy pay-to-unlock flow
		app.info.Printf(lm.StripePaymentOldInvite, refID)
		if inv, ok := app.storage.GetInvitesKey(refID); ok {
			inv.PaymentStatus = "paid"
			app.storage.SetInvitesKey(refID, inv)
		}
		return
	}

	app.info.Printf(lm.StripePaymentReceived, metadata["plan"], targetEmail)

	existingUserID, existingEmail, found := app.findUserByEmail(targetEmail)
	if found {
		app.info.Printf(lm.ExistingUserFound, targetEmail, existingUserID)

		if subscriptionID := session.Subscription; subscriptionID != nil {
			existingEmail.Label = "Stripe Subscription: " + subscriptionID.ID
		} else {
			existingEmail.Label = "Purchased via Store"
		}
		app.storage.SetEmailsKey(existingUserID, existingEmail)

		expiry := time.Now()
		lastTx := ""
		if userExpiry, ok := app.storage.GetUserExpiryKey(existingUserID); ok {
			expiry = userExpiry.Expiry
			lastTx = userExpiry.LastTransactionID
		}

		if lastTx == session.ID {
			app.info.Printf(lm.StripeSessionAlreadyProcessed, session.ID, existingUserID)
			return
		}

		if expiry.Before(time.Now()) {
			expiry = time.Now()
		}
		var newExpiry time.Time
		if metadata["plan"] == "Monthly" {
			newExpiry = expiry.AddDate(0, 1, 0)
		} else {
			newExpiry = expiry.AddDate(10, 0, 0)
		}
		app.storage.SetUserExpiryKey(existingUserID, UserExpiry{Expiry: newExpiry, LastTransactionID: session.ID})

		if paramsUser, err := app.jf.UserByID(existingUserID, false); err == nil {
			if err, _, _ = app.SetUserDisabled(paramsUser, false); err != nil {
				app.err.Printf(lm.FailedReEnableUser, existingUserID, err)
			}
			app.InvalidateUserCaches()
		}

		app.info.Printf(lm.UserReactivated, existingUserID, newExpiry)
		return
	}

	// New user: generate invite
	inviteCode := GenerateInviteCode()
	profile := metadata["profile"]
	if profile == "" {
		profile = "Default"
	}
	if _, ok := app.storage.GetProfileKey(profile); !ok {
		app.debug.Printf(lm.FailedGetProfile, profile)
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
		invite.ValidTill = time.Now().AddDate(0, 1, 0)
		invite.UserExpiry = true
		invite.UserMonths = 1
	} else {
		invite.ValidTill = time.Now().AddDate(10, 0, 0)
		invite.UserExpiry = false
	}

	app.storage.SetInvitesKey(inviteCode, invite)
	app.info.Printf(lm.GeneratedInviteForPurchase, inviteCode, targetEmail)

	if emailEnabled {
		go func(inv Invite, tEmail string) {
			msg, err := app.email.constructInvite(&inv, false)
			if err != nil {
				app.err.Printf(lm.FailedConstructInviteMessage, tEmail, err)
				return
			}
			if err = app.email.send(msg, tEmail); err != nil {
				app.err.Printf(lm.FailedSendInviteMessage, inv.Code, tEmail, err)
			} else {
				app.info.Printf(lm.SentInviteMessage, inv.Code, tEmail)
			}
		}(invite, targetEmail)
	}
}

func (app *appContext) handleStripeInvoiceSucceeded(event *stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	if invoice.BillingReason != stripe.InvoiceBillingReasonSubscriptionCycle {
		return
	}
	if invoice.CustomerEmail == "" {
		app.err.Printf(lm.FailedFindUserByEmail, fmt.Sprintf("invoice %s (no email)", invoice.ID))
		return
	}

	email := invoice.CustomerEmail
	app.info.Printf(lm.StripeRenewalReceived, email)

	userID, _, found := app.findUserByEmail(email)
	if !found {
		app.err.Printf(lm.FailedFindUserByEmail, email)
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

	app.info.Printf(lm.UserExpiryExtended, userID, email, newExpiry)
}

func (app *appContext) handleStripeSubscriptionDeleted(event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	targetEmail := sub.Metadata["target_email"]
	if targetEmail == "" {
		app.debug.Printf(lm.StripeSubscriptionDeleted, sub.ID, "unknown (no metadata)")
		return
	}

	app.info.Printf(lm.StripeSubscriptionDeleted, sub.ID, targetEmail)

	userID, _, found := app.findUserByEmail(targetEmail)
	if !found {
		app.err.Printf(lm.FailedFindUserByEmail, targetEmail)
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
