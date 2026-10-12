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
	"github.com/stripe/stripe-go/v86"
	chargeapi "github.com/stripe/stripe-go/v86/charge"
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
	lockToken, lockHash, err := newPaymentLockToken()
	if err != nil {
		app.err.Printf("Failed to create payment lock: %v", err)
		respond(500, "Failed to create payment lock", gc)
		return
	}

	metadata := app.stripePaymentMetadata(map[string]string{
		stripeMetadataFlow:       stripeMetadataFlowInviteUnlock,
		stripeMetadataInviteCode: code,
		stripeMetadataEmail:      inv.SendTo,
		stripeMetadataPlan:       "Invite",
		stripeMetadataProfile:    inv.Profile,
	})

	session, err := CreateCheckoutSession(code, inv.PriceAmount, inv.PriceCurrency, "Invite Code: "+code, successURL, cancelURL, metadata, "", 0)
	if err != nil {
		app.err.Printf(lm.FailedCreateCheckoutSession, err)
		respond(500, "Failed to create checkout session", gc)
		return
	}

	app.setPayment(session.ID, func(payment *Payment) {
		payment.Provider = lm.Stripe
		payment.InstanceID = metadata[stripeMetadataInstanceID]
		payment.ProviderPaymentID = session.ID
		payment.ProviderLiveMode = session.Livemode
		if session.Customer != nil {
			payment.CustomerID = session.Customer.ID
		}
		payment.TargetEmail = inv.SendTo
		payment.Plan = "Invite"
		payment.Profile = inv.Profile
		payment.Amount = inv.PriceAmount
		payment.Currency = inv.PriceCurrency
		payment.Status = paymentStatusCheckoutCreated
		payment.EmailStatus = paymentEmailNotApplicable
		payment.InviteCode = code
		payment.InviteLockHash = lockHash
		setPaymentLockCreated(payment)
		if session.Created > 0 {
			payment.Created = time.Unix(session.Created, 0)
		}
	})

	gc.SetCookie(paymentLockCookieName, lockToken, paymentLockMaxAge, "/", "", false, true)

	gc.JSON(200, stringResponse{Response: session.URL})
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

	plan, ok := app.paymentPlanByID(req.Plan)
	if !ok || !plan.Enabled {
		respond(400, "Invalid payment plan", gc)
		return
	}

	// Double-billing prevention for subscription plans
	if plan.Recurring {
		if userID, active := app.userHasActivePaidExpiry(req.Email); active {
			app.info.Printf(lm.StripeBlockedDuplicate, userID, req.Email)
			respond(409, "You already have an active subscription.", gc)
			return
		}
	}

	refID := "purchase-" + strconv.FormatInt(time.Now().Unix(), 10)

	baseURL := ExternalURI(gc)
	successURL := fmt.Sprintf("%s/payment/success", baseURL)
	cancelURL := fmt.Sprintf("%s/store?canceled=true", baseURL)

	metadata := app.stripePaymentMetadata(plan.metadata())
	metadata[stripeMetadataFlow] = stripeMetadataFlowStorePurchase
	metadata[stripeMetadataEmail] = req.Email

	session, err := CreateCheckoutSession(refID, plan.Price, plan.Currency, plan.Name, successURL, cancelURL, metadata, plan.StripeInterval, plan.StripeIntervalCount)
	if err != nil {
		app.err.Printf(lm.FailedCreateCheckoutSession, err)
		respond(500, "Failed to create checkout session", gc)
		return
	}

	app.setPayment(session.ID, func(payment *Payment) {
		payment.Provider = lm.Stripe
		payment.InstanceID = metadata[stripeMetadataInstanceID]
		payment.ProviderPaymentID = session.ID
		payment.ProviderLiveMode = session.Livemode
		if session.Customer != nil {
			payment.CustomerID = session.Customer.ID
		}
		payment.TargetEmail = req.Email
		paymentPlanSnapshotFromMetadata(metadata).apply(payment)
		payment.Amount = plan.Price
		payment.Currency = plan.Currency
		payment.Status = paymentStatusCheckoutCreated
		payment.EmailStatus = paymentEmailNotStarted
		if session.Created > 0 {
			payment.Created = time.Unix(session.Created, 0)
		}
	})

	gc.JSON(200, stringResponse{Response: session.URL})
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
	verifySignature := app.config.Section("stripe").Key("verify_signature").MustBool(true)

	if !verifySignature {
		app.debug.Println(lm.StripeSignatureBypass)
	}

	event, err := HandleWebhook(payload, sigHeader, webhookSecret, verifySignature)
	if err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		gc.AbortWithStatus(400)
		return
	}
	app.info.Printf("Stripe webhook received: %s (%s)", event.Type, event.ID)

	switch event.Type {
	case "checkout.session.completed":
		app.handleStripeCheckoutCompleted(event)
	case "checkout.session.expired":
		app.handleStripeCheckoutExpired(event)
	case "invoice.payment_succeeded":
		app.handleStripeInvoiceSucceeded(event)
	case "invoice.payment_failed", "invoice.marked_uncollectible", "invoice.voided":
		app.handleStripeInvoicePaymentFailed(event)
	case "payment_intent.canceled", "payment_intent.payment_failed", "payment_intent.succeeded":
		app.handleStripePaymentIntentUpdated(event)
	case "charge.refunded", "charge.updated", "charge.failed":
		app.handleStripeChargeUpdated(event)
	case "refund.created", "refund.updated":
		app.handleStripeRefundUpdated(event)
	case "customer.subscription.updated":
		app.handleStripeSubscriptionUpdated(event)
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
	if metadata == nil {
		metadata = map[string]string{}
	}
	subscriptionID := ""
	if session.Subscription != nil {
		subscriptionID = session.Subscription.ID
	}

	app.setPayment(session.ID, func(payment *Payment) {
		applyStripeSessionToPayment(firstNonEmpty(metadata[stripeMetadataInstanceID], app.paymentInstanceID()), &session, payment)
		paymentPlanSnapshotFromMetadata(metadata).apply(payment)
		if subscriptionID != "" {
			payment.SubscriptionID = subscriptionID
		}
		payment.Status = paymentStatusPaid
		payment.PaidAt = time.Now()
	})

	if stripeSessionIsInviteUnlock(metadata) {
		if app.fulfillStripeInviteUnlock(session.ID, refID, metadata) {
			return
		}
		app.setPayment(session.ID, func(payment *Payment) {
			payment.Status = paymentStatusNeedsReview
			payment.Error = "Stripe invite unlock is paid, but the local invite could not be found"
		})
		return
	}

	targetEmail, ok := metadata[stripeMetadataEmail]
	if !ok {
		// Legacy pay-to-unlock flow
		app.fulfillStripeInviteUnlock(session.ID, refID, metadata)
		return
	}

	snapshot := paymentPlanSnapshotFromMetadata(metadata)
	plan := snapshot.Name
	app.info.Printf(lm.StripePaymentReceived, plan, targetEmail)

	app.setPayment(session.ID, func(payment *Payment) {
		payment.TargetEmail = targetEmail
		snapshot.apply(payment)
	})

	result := app.fulfillStorePayment(paymentFulfillment{
		Provider:            lm.Stripe,
		TransactionID:       session.ID,
		SubscriptionID:      subscriptionID,
		TargetEmail:         targetEmail,
		PlanID:              snapshot.ID,
		Plan:                plan,
		Profile:             snapshot.Profile,
		AccessMonths:        snapshot.AccessMonths,
		AccessDays:          snapshot.AccessDays,
		Recurring:           snapshot.Recurring,
		StripeInterval:      snapshot.StripeInterval,
		StripeIntervalCount: snapshot.StripeIntervalCount,
	})
	app.markPaymentFulfilled(session.ID, result)
	if result.JellyfinID != "" && !result.Duplicate {
		app.sendStorePaymentConfirmation(result.JellyfinID, targetEmail, session.ID, lm.Stripe, plan, result.Expiry, snapshot.Recurring)
	}
	if result.ShouldSendInvite {
		app.sendPurchasedInvite(result.Invite, targetEmail, session.ID, plan)
	} else if result.InviteCode != "" {
		app.markPaymentEmail(session.ID, paymentEmailDisabled, "")
	}
}

func (app *appContext) handleStripeCheckoutExpired(event *stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	app.setPayment(session.ID, func(payment *Payment) {
		applyStripeSessionToPayment(firstNonEmpty(session.Metadata[stripeMetadataInstanceID], app.paymentInstanceID()), &session, payment)
		setPaymentLifecycleStatus(payment, paymentStatusCheckoutExpired, "Stripe checkout session expired")
	})
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

	subscriptionID := ""
	instanceID := app.paymentInstanceID()
	metadata := map[string]string{}
	if invoice.Parent != nil &&
		invoice.Parent.SubscriptionDetails != nil &&
		invoice.Parent.SubscriptionDetails.Subscription != nil {
		subscriptionID = invoice.Parent.SubscriptionDetails.Subscription.ID
		if invoice.Parent.SubscriptionDetails.Metadata != nil {
			metadata = invoice.Parent.SubscriptionDetails.Metadata
			if metadataInstanceID := invoice.Parent.SubscriptionDetails.Metadata[stripeMetadataInstanceID]; metadataInstanceID != "" {
				instanceID = metadataInstanceID
			}
		}
	}
	planSnapshot := app.paymentPlanSnapshotForSubscription(subscriptionID, metadata)
	app.setPayment(invoice.ID, func(payment *Payment) {
		payment.Provider = lm.Stripe
		payment.InstanceID = instanceID
		payment.ProviderPaymentID = invoice.ID
		applyStripeInvoiceToPayment(&invoice, payment)
		planSnapshot.apply(payment)
		payment.SubscriptionID = subscriptionID
		payment.TargetEmail = invoice.CustomerEmail
		if payment.Plan == "" {
			payment.Plan = paymentPlanMonthly
		}
		payment.Status = paymentStatusPaid
		payment.EmailStatus = paymentEmailNotApplicable
		if invoice.Created > 0 {
			created := time.Unix(invoice.Created, 0)
			payment.Created = created
			payment.PaidAt = created
		} else {
			payment.PaidAt = time.Now()
		}
	})

	if invoice.CustomerEmail == "" {
		app.markPaymentError(invoice.ID, fmt.Sprintf("invoice %s has no customer email", invoice.ID))
		app.err.Printf(lm.FailedFindUserByEmail, fmt.Sprintf("invoice %s (no email)", invoice.ID))
		return
	}

	email := invoice.CustomerEmail
	app.info.Printf(lm.StripeRenewalReceived, email)

	userID, _, found := app.findUserByEmail(email)
	if !found {
		app.markPaymentError(invoice.ID, fmt.Sprintf("could not find user with email %s", email))
		app.err.Printf(lm.FailedFindUserByEmail, email)
		return
	}

	newExpiry, changed := app.extendPaidUserExpiry(userID, invoice.ID, planSnapshot.Name, planSnapshot.AccessMonths, planSnapshot.AccessDays)
	if !changed {
		app.info.Printf(lm.PaymentTransactionAlreadyProcessed, lm.Stripe, invoice.ID, userID)
		app.markPaymentFulfilled(invoice.ID, paymentFulfillmentResult{JellyfinID: userID})
		return
	}
	app.reEnablePaidUser(userID)
	app.markPaymentFulfilled(invoice.ID, paymentFulfillmentResult{JellyfinID: userID})
	app.sendStorePaymentConfirmation(userID, email, invoice.ID, lm.Stripe, planSnapshot.Name, newExpiry, planSnapshot.Recurring)

	app.info.Printf(lm.UserExpiryExtended, userID, email, newExpiry)
}

func (app *appContext) handleStripeInvoicePaymentFailed(event *stripe.Event) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	subscriptionID := stripeInvoiceSubscriptionID(&invoice)
	instanceID := app.paymentInstanceID()
	metadata := map[string]string{}
	if invoice.Parent != nil && invoice.Parent.SubscriptionDetails != nil && invoice.Parent.SubscriptionDetails.Metadata != nil {
		metadata = invoice.Parent.SubscriptionDetails.Metadata
		if metadataInstanceID := invoice.Parent.SubscriptionDetails.Metadata[stripeMetadataInstanceID]; metadataInstanceID != "" {
			instanceID = metadataInstanceID
		}
	}
	planSnapshot := app.paymentPlanSnapshotForSubscription(subscriptionID, metadata)

	app.setPayment(invoice.ID, func(payment *Payment) {
		payment.Provider = lm.Stripe
		payment.InstanceID = instanceID
		payment.ProviderPaymentID = invoice.ID
		payment.EmailStatus = paymentEmailNotApplicable
		planSnapshot.apply(payment)
		if payment.Plan == "" {
			payment.Plan = paymentPlanMonthly
		}
		applyStripeInvoiceToPayment(&invoice, payment)
		if subscriptionID != "" {
			payment.SubscriptionID = subscriptionID
		}
	})
	if subscriptionID != "" {
		app.setPaymentsByStripeIDs("", "", "", subscriptionID, func(payment *Payment) {
			applyStripeInvoiceToPayment(&invoice, payment)
		})
	}
}

func (app *appContext) handleStripePaymentIntentUpdated(event *stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	chargeID := stripePaymentIntentChargeID(&paymentIntent)
	app.setPaymentsByStripeIDs(paymentIntent.ID, chargeID, "", "", func(payment *Payment) {
		applyStripePaymentIntentToPayment(&paymentIntent, payment)
	})
}

func (app *appContext) handleStripeChargeUpdated(event *stripe.Event) {
	var charge stripe.Charge
	if err := json.Unmarshal(event.Data.Raw, &charge); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	app.applyStripeChargeEvent(&charge)
}

func (app *appContext) handleStripeRefundUpdated(event *stripe.Event) {
	var refund stripe.Refund
	if err := json.Unmarshal(event.Data.Raw, &refund); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	chargeID := ""
	if refund.Charge != nil {
		chargeID = refund.Charge.ID
	}
	if chargeID != "" {
		if charge, err := chargeapi.Get(chargeID, nil); err == nil {
			app.applyStripeChargeEvent(charge)
			return
		}
	}

	paymentIntentID := ""
	if refund.PaymentIntent != nil {
		paymentIntentID = refund.PaymentIntent.ID
	}
	app.setPaymentsByStripeIDs(paymentIntentID, chargeID, "", "", func(payment *Payment) {
		applyStripeRefundToPayment(&refund, payment)
	})
}

func (app *appContext) applyStripeChargeEvent(charge *stripe.Charge) {
	if charge == nil {
		return
	}
	paymentIntentID := ""
	if charge.PaymentIntent != nil {
		paymentIntentID = charge.PaymentIntent.ID
	}
	app.setPaymentsByStripeIDs(paymentIntentID, charge.ID, "", "", func(payment *Payment) {
		applyStripeChargeToPayment(charge, payment)
	})
}

func applyStripeRefundToPayment(refund *stripe.Refund, payment *Payment) {
	if refund == nil || refund.ID == "" {
		return
	}
	if refund.PaymentIntent != nil {
		payment.PaymentIntentID = refund.PaymentIntent.ID
	}
	if refund.Charge != nil {
		payment.ChargeID = refund.Charge.ID
	}
	if refund.Currency != "" {
		payment.Currency = string(refund.Currency)
	}
	if refund.Status != stripe.RefundStatusSucceeded && refund.Status != stripe.RefundStatusPending {
		return
	}
	if refund.Amount > payment.RefundedAmount {
		payment.RefundedAmount = refund.Amount
	}
	if payment.Amount > 0 && payment.RefundedAmount >= payment.Amount {
		setPaymentLifecycleStatus(payment, paymentStatusRefunded, "")
		return
	}
	setPaymentLifecycleStatus(payment, paymentStatusPartiallyRefunded, "")
}

func (app *appContext) handleStripeSubscriptionUpdated(event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	app.applyStripeSubscriptionEvent(&sub)
	if sub.CancelAtPeriodEnd || sub.Status == stripe.SubscriptionStatusCanceled {
		app.notifyStripeSubscriptionCancellation(&sub, "stripe")
	}
	if sub.Status == stripe.SubscriptionStatusCanceled {
		app.expireStripeSubscriptionUser(&sub)
	}
}

func (app *appContext) handleStripeSubscriptionDeleted(event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		app.err.Printf(lm.StripeWebhookError, err)
		return
	}

	app.applyStripeSubscriptionEvent(&sub)
	app.notifyStripeSubscriptionCancellation(&sub, "stripe")
	targetEmail := stripeSubscriptionTargetEmail(&sub, app)
	if targetEmail == "" {
		app.debug.Printf(lm.StripeSubscriptionDeleted, sub.ID, "unknown (no metadata)")
		return
	}

	app.info.Printf(lm.StripeSubscriptionDeleted, sub.ID, targetEmail)
	app.expireStripeSubscriptionUser(&sub)
}

func (app *appContext) applyStripeSubscriptionEvent(sub *stripe.Subscription) {
	if sub == nil || sub.ID == "" {
		return
	}
	found := app.setPaymentsByStripeIDs("", "", "", sub.ID, func(payment *Payment) {
		applyStripeSubscriptionToPayment(sub, payment)
	})
	if found || !stripeSubscriptionMatchesInstance(app.paymentInstanceID(), sub) {
		return
	}
	targetEmail := stripeSubscriptionTargetEmail(sub, app)
	if targetEmail == "" {
		return
	}
	app.setPayment("subscription-"+sub.ID, func(payment *Payment) {
		payment.Provider = lm.Stripe
		payment.InstanceID = app.paymentInstanceID()
		payment.ProviderPaymentID = sub.ID
		payment.EmailStatus = paymentEmailNotApplicable
		app.paymentPlanSnapshotForSubscription(sub.ID, sub.Metadata).apply(payment)
		if payment.Plan == "" {
			payment.Plan = paymentPlanMonthly
		}
		applyStripeSubscriptionToPayment(sub, payment)
	})
}

func (app *appContext) expireStripeSubscriptionUser(sub *stripe.Subscription) {
	targetEmail := stripeSubscriptionTargetEmail(sub, app)
	if targetEmail == "" {
		return
	}
	userID, _, found := app.findUserByEmail(targetEmail)
	if !found {
		app.err.Printf(lm.FailedFindUserByEmail, targetEmail)
		return
	}

	app.expirePaidUserNow(userID)

	if paramsUser, err := app.jf.UserByID(userID, false); err == nil {
		if err, _, _ = app.SetUserDisabled(paramsUser, true); err != nil {
			app.err.Printf(lm.FailedDisableUser, userID, err)
		} else {
			app.info.Printf(lm.UserDisabledDueToCancellation, userID)
		}
		app.InvalidateUserCaches()
	}
}

func stripeSubscriptionTargetEmail(sub *stripe.Subscription, app *appContext) string {
	if sub != nil && sub.Metadata != nil && sub.Metadata[stripeMetadataEmail] != "" {
		return sub.Metadata[stripeMetadataEmail]
	}
	if sub != nil {
		for _, payment := range app.storage.GetPayments() {
			if payment.SubscriptionID == sub.ID && payment.TargetEmail != "" {
				return payment.TargetEmail
			}
		}
	}
	return ""
}

func stripeSubscriptionMatchesInstance(instanceID string, sub *stripe.Subscription) bool {
	if sub == nil || sub.Metadata == nil {
		return false
	}
	return sub.Metadata[stripeMetadataSource] == stripeMetadataSourceJFA &&
		sub.Metadata[stripeMetadataInstanceID] == instanceID
}
