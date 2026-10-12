package main

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/lithammer/shortuuid/v3"
	"github.com/stripe/stripe-go/v86"
	chargeapi "github.com/stripe/stripe-go/v86/charge"
	"github.com/stripe/stripe-go/v86/checkout/session"
	invoiceapi "github.com/stripe/stripe-go/v86/invoice"
	invoicepaymentapi "github.com/stripe/stripe-go/v86/invoicepayment"
	paymentintentapi "github.com/stripe/stripe-go/v86/paymentintent"
	subscriptionapi "github.com/stripe/stripe-go/v86/subscription"
	"gopkg.in/ini.v1"
)

const (
	stripeMetadataSource        = "source"
	stripeMetadataSourceJFA     = "jfa-go"
	stripeMetadataInstanceID    = "instance_id"
	stripeMetadataEmail         = "target_email"
	stripeMetadataPlanID        = "plan_id"
	stripeMetadataPlan          = "plan"
	stripeMetadataProfile       = "profile"
	stripeMetadataAccessMonths  = "access_months"
	stripeMetadataAccessDays    = "access_days"
	stripeMetadataRecurring     = "recurring"
	stripeMetadataInterval      = "interval"
	stripeMetadataIntervalCount = "interval_count"
	stripeMetadataFlow          = "flow"
	stripeMetadataInviteCode    = "invite_code"

	stripeMetadataFlowInviteUnlock  = "invite_unlock"
	stripeMetadataFlowStorePurchase = "store_purchase"

	stripeReconcileLookbackDays = 30
)

func (app *appContext) paymentInstanceID() string {
	id := strings.TrimSpace(app.config.Section("stripe").Key("instance_id").String())
	if id != "" {
		return id
	}

	id = "jfa_" + shortuuid.New()
	app.config.Section("stripe").Key("instance_id").SetValue(id)
	if app.configPath == "" {
		return id
	}

	tempConfig, err := ini.ShadowLoad(app.configPath)
	if err != nil {
		if app.err != nil {
			app.err.Printf(lm.FailedLoadConfig, app.configPath, err)
		}
		return id
	}
	tempConfig.Section("stripe").Key("instance_id").SetValue(id)
	if err = tempConfig.SaveTo(app.configPath); err != nil && app.err != nil {
		app.err.Printf(lm.FailedWriting, app.configPath, err)
	}
	return id
}

func (app *appContext) stripePaymentMetadata(values map[string]string) map[string]string {
	metadata := map[string]string{
		stripeMetadataSource:     stripeMetadataSourceJFA,
		stripeMetadataInstanceID: app.paymentInstanceID(),
	}
	for k, v := range values {
		if v != "" {
			metadata[k] = v
		}
	}
	return metadata
}

func (app *appContext) ReconcileStripePayments(gc *gin.Context) {
	if !stripeEnabled {
		respond(400, "Stripe disabled", gc)
		return
	}

	result := app.reconcileStripePayments()
	if result.Error != "" {
		gc.JSON(500, result)
		return
	}
	gc.JSON(200, result)
}

func (app *appContext) reconcileStripePayments() ReconcilePaymentsDTO {
	instanceID := app.paymentInstanceID()
	params := &stripe.CheckoutSessionListParams{
		CreatedRange: &stripe.RangeQueryParams{
			GreaterThanOrEqual: time.Now().AddDate(0, 0, -stripeReconcileLookbackDays).Unix(),
		},
	}
	params.Limit = stripe.Int64(100)
	params.AddExpand("data.payment_intent.latest_charge")
	params.AddExpand("data.subscription")
	params.AddExpand("data.subscription.latest_invoice")
	params.AddExpand("data.invoice")

	sessions := []*stripe.CheckoutSession{}
	iter := session.List(params)
	for iter.Next() {
		sessions = append(sessions, iter.CheckoutSession())
	}
	if err := iter.Err(); err != nil {
		return ReconcilePaymentsDTO{Error: err.Error()}
	}

	result := app.reconcileStripeCheckoutSessions(instanceID, sessions)
	app.reconcileStoredStripePayments(instanceID, &result)
	return result
}

func (app *appContext) reconcileStripeCheckoutSessions(instanceID string, sessions []*stripe.CheckoutSession) ReconcilePaymentsDTO {
	result := ReconcilePaymentsDTO{}
	for _, session := range sessions {
		result.Scanned++
		if !stripeSessionMatchesInstance(instanceID, session) {
			result.Skipped++
			continue
		}
		result.Matched++
		created, needsReview, lifecycleUpdated := app.reconcileStripeCheckoutSession(instanceID, session)
		if created {
			result.Created++
		} else {
			result.Updated++
		}
		if needsReview {
			result.NeedsReview++
		}
		if lifecycleUpdated {
			result.LifecycleUpdates++
		}
	}
	return result
}

func stripeSessionMatchesInstance(instanceID string, session *stripe.CheckoutSession) bool {
	if session == nil || session.Metadata == nil {
		return false
	}
	return session.Metadata[stripeMetadataSource] == stripeMetadataSourceJFA &&
		session.Metadata[stripeMetadataInstanceID] == instanceID
}

func (app *appContext) reconcileStripeCheckoutSession(instanceID string, session *stripe.CheckoutSession) (bool, bool, bool) {
	_, existed := app.storage.GetPaymentKey(session.ID)
	needsReview := false
	lifecycleUpdated := false

	app.setPayment(session.ID, func(payment *Payment) {
		previousStatus := payment.Status
		applyStripeSessionToPayment(instanceID, session, payment)
		app.recoverPaymentLocalLink(payment)
		if !app.paymentHasRecoverableLocalLink(payment) && stripeSessionPaid(session) {
			payment.Status = paymentStatusNeedsReview
			payment.Error = "Stripe checkout is paid, but no local invite or Jellyfin user link could be recovered"
			needsReview = true
		}
		lifecycleUpdated = previousStatus != payment.Status && paymentStatusIsLifecycle(payment.Status)
	})
	if app.fulfillRecoveredStripeCheckout(session) {
		needsReview = false
	}

	return !existed, needsReview, lifecycleUpdated
}

func (app *appContext) reconcileStoredStripePayments(instanceID string, result *ReconcilePaymentsDTO) {
	for _, payment := range app.storage.GetPayments() {
		if payment.Provider != lm.Stripe || (payment.InstanceID != "" && payment.InstanceID != instanceID) {
			continue
		}
		refreshed, needsReview, lifecycleUpdated := app.reconcileStoredStripePayment(instanceID, payment)
		if !refreshed {
			result.Skipped++
			continue
		}
		result.Refreshed++
		result.Updated++
		if needsReview {
			result.NeedsReview++
		}
		if lifecycleUpdated {
			result.LifecycleUpdates++
		}
	}
}

func (app *appContext) reconcileStoredStripePayment(instanceID string, stored Payment) (bool, bool, bool) {
	state := stripePaymentState{}
	state.load(app, stored)

	needsReview := false
	lifecycleUpdated := false
	app.setPayment(stored.ID, func(payment *Payment) {
		previousStatus := payment.Status
		payment.LastReconciledAt = time.Now()

		if state.session != nil {
			applyStripeSessionToPayment(instanceID, state.session, payment)
		}
		if state.invoice != nil {
			applyStripeInvoiceToPayment(state.invoice, payment)
		}
		if state.subscription != nil {
			applyStripeSubscriptionToPayment(state.subscription, payment)
		}
		if state.paymentIntent != nil {
			applyStripePaymentIntentToPayment(state.paymentIntent, payment)
		}
		if state.charge != nil {
			applyStripeChargeToPayment(state.charge, payment)
		}

		app.recoverPaymentLocalLink(payment)
		if !app.paymentHasRecoverableLocalLink(payment) && !payment.PaidAt.IsZero() && !paymentStatusIsLifecycle(payment.Status) {
			payment.Status = paymentStatusNeedsReview
			payment.Error = "Stripe payment is paid, but no local invite or Jellyfin user link could be recovered"
			needsReview = true
		}
		if state.errText != "" && payment.Error == "" {
			payment.Error = state.errText
		}
		lifecycleUpdated = previousStatus != payment.Status && paymentStatusIsLifecycle(payment.Status)
	})
	if app.fulfillRecoveredStripeCheckout(state.session) {
		needsReview = false
	}

	return state.loaded(), needsReview, lifecycleUpdated
}

func (app *appContext) fulfillRecoveredStripeCheckout(session *stripe.CheckoutSession) bool {
	if session == nil || session.ID == "" || !stripeSessionPaid(session) {
		return false
	}
	payment, ok := app.storage.GetPaymentKey(session.ID)
	if ok && !payment.FulfilledAt.IsZero() {
		if app.repairStaleStorePaymentAsInvite(payment) {
			return true
		}
		if shouldSendStorePaymentConfirmation(payment) {
			expiry := time.Time{}
			if userExpiry, ok := app.storage.GetUserExpiryKey(payment.JellyfinID); ok {
				expiry = userExpiry.Expiry
			}
			app.sendStorePaymentConfirmation(payment.JellyfinID, payment.TargetEmail, payment.ID, payment.Provider, payment.Plan, expiry, payment.Recurring)
		}
		return true
	}

	metadata := session.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}

	refID := session.ClientReferenceID
	targetEmail := metadata[stripeMetadataEmail]
	if stripeSessionIsInviteUnlock(metadata) || targetEmail == "" {
		return app.fulfillStripeInviteUnlock(session.ID, refID, metadata)
	}

	subscriptionID := stripeSessionSubscriptionID(session)
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
	} else if result.JellyfinID != "" && result.Duplicate {
		app.markPaymentEmail(session.ID, paymentEmailNotApplicable, "")
	}
	return result.InviteCode != "" || result.JellyfinID != ""
}

func stripeSessionIsInviteUnlock(metadata map[string]string) bool {
	if metadata == nil {
		return false
	}
	return metadata[stripeMetadataFlow] == stripeMetadataFlowInviteUnlock || metadata[stripeMetadataInviteCode] != ""
}

func (app *appContext) fulfillStripeInviteUnlock(paymentID, refID string, metadata map[string]string) bool {
	inviteCode := ""
	if metadata != nil {
		inviteCode = metadata[stripeMetadataInviteCode]
	}
	if inviteCode == "" {
		inviteCode = refID
	}
	if inviteCode == "" {
		return false
	}

	app.info.Printf(lm.StripePaymentOldInvite, inviteCode)
	inv, ok := app.storage.GetInvitesKey(inviteCode)
	if !ok {
		return false
	}
	inv.PaymentID = paymentID
	inv.PaymentStatus = "paid"
	app.storage.SetInvitesKey(inviteCode, inv)
	app.markPaymentFulfilled(paymentID, paymentFulfillmentResult{InviteCode: inviteCode})
	app.markPaymentEmail(paymentID, paymentEmailNotApplicable, "")
	return true
}

type stripePaymentState struct {
	session       *stripe.CheckoutSession
	invoice       *stripe.Invoice
	subscription  *stripe.Subscription
	paymentIntent *stripe.PaymentIntent
	charge        *stripe.Charge
	errText       string
}

func (s *stripePaymentState) loaded() bool {
	return s.session != nil || s.invoice != nil || s.subscription != nil || s.paymentIntent != nil || s.charge != nil || s.errText != ""
}

func (s *stripePaymentState) load(app *appContext, stored Payment) {
	stripeID := firstNonEmpty(stored.ProviderPaymentID, stored.ID)
	if strings.HasPrefix(stripeID, "cs_") {
		params := &stripe.CheckoutSessionParams{}
		params.AddExpand("payment_intent.latest_charge")
		params.AddExpand("subscription")
		params.AddExpand("subscription.latest_invoice")
		params.AddExpand("invoice")
		checkoutSession, err := session.Get(stripeID, params)
		if err != nil {
			s.addErr(err)
		} else {
			s.session = checkoutSession
		}
	}

	invoiceID := firstNonEmpty(stored.InvoiceID, stripeSessionInvoiceID(s.session))
	if invoiceID == "" && strings.HasPrefix(stripeID, "in_") {
		invoiceID = stripeID
	}
	if invoiceID != "" {
		params := &stripe.InvoiceParams{}
		params.AddExpand("parent.subscription_details.subscription")
		inv, err := invoiceapi.Get(invoiceID, params)
		if err != nil {
			s.addErr(err)
		} else {
			s.invoice = inv
			s.loadInvoicePayment(inv.ID)
		}
	}

	subscriptionID := firstNonEmpty(stored.SubscriptionID, stripeSessionSubscriptionID(s.session), stripeInvoiceSubscriptionID(s.invoice))
	if subscriptionID == "" && strings.HasPrefix(stripeID, "sub_") {
		subscriptionID = stripeID
	}
	if subscriptionID != "" {
		params := &stripe.SubscriptionParams{}
		params.AddExpand("latest_invoice")
		sub, err := subscriptionapi.Get(subscriptionID, params)
		if err != nil {
			s.addErr(err)
		} else {
			s.subscription = sub
			if s.invoice == nil && sub.LatestInvoice != nil {
				s.invoice = sub.LatestInvoice
				s.loadInvoicePayment(sub.LatestInvoice.ID)
			}
		}
	}

	paymentIntentID := firstNonEmpty(stored.PaymentIntentID, stripeSessionPaymentIntentID(s.session), stripeInvoicePaymentIntentID(s.invoice))
	if paymentIntentID == "" && strings.HasPrefix(stripeID, "pi_") {
		paymentIntentID = stripeID
	}
	if paymentIntentID != "" && (s.paymentIntent == nil || s.paymentIntent.ID != paymentIntentID) {
		params := &stripe.PaymentIntentParams{}
		params.AddExpand("latest_charge")
		pi, err := paymentintentapi.Get(paymentIntentID, params)
		if err != nil {
			s.addErr(err)
		} else {
			s.paymentIntent = pi
		}
	}

	chargeID := firstNonEmpty(stored.ChargeID, stripePaymentIntentChargeID(s.paymentIntent))
	if chargeID == "" && strings.HasPrefix(stripeID, "ch_") {
		chargeID = stripeID
	}
	if chargeID != "" && (s.charge == nil || s.charge.ID != chargeID) {
		ch, err := chargeapi.Get(chargeID, nil)
		if err != nil {
			s.addErr(err)
		} else {
			s.charge = ch
		}
	}

	if app.debug != nil && s.errText != "" {
		app.debug.Printf("Stripe reconciliation warning for %s: %s", stored.ID, s.errText)
	}
}

func (s *stripePaymentState) loadInvoicePayment(invoiceID string) {
	if invoiceID == "" {
		return
	}

	params := &stripe.InvoicePaymentListParams{Invoice: stripe.String(invoiceID)}
	params.Limit = stripe.Int64(10)
	params.AddExpand("data.payment.payment_intent.latest_charge")
	params.AddExpand("data.payment.charge")
	iter := invoicepaymentapi.List(params)
	for iter.Next() {
		ip := iter.InvoicePayment()
		if ip.Payment == nil {
			continue
		}
		if ip.Payment.PaymentIntent != nil {
			s.paymentIntent = ip.Payment.PaymentIntent
		}
		if ip.Payment.Charge != nil {
			s.charge = ip.Payment.Charge
		}
	}
	if err := iter.Err(); err != nil {
		s.addErr(err)
	}
}

func (s *stripePaymentState) addErr(err error) {
	if err == nil {
		return
	}
	if s.errText == "" {
		s.errText = "Stripe reconciliation: " + err.Error()
		return
	}
	s.errText += "; " + err.Error()
}

func (app *appContext) recoverPaymentLocalLink(payment *Payment) {
	if payment.JellyfinID != "" || payment.TargetEmail == "" {
		return
	}
	if userID, _, ok := app.findUserByEmail(payment.TargetEmail); ok {
		payment.JellyfinID = userID
	}
}

func (app *appContext) paymentHasRecoverableLocalLink(payment *Payment) bool {
	if payment.JellyfinID != "" {
		return true
	}
	if payment.InviteCode == "" {
		return false
	}
	_, ok := app.storage.GetInvitesKey(payment.InviteCode)
	return ok
}

func applyStripeSessionToPayment(instanceID string, session *stripe.CheckoutSession, payment *Payment) {
	payment.Provider = lm.Stripe
	payment.InstanceID = instanceID
	payment.ProviderPaymentID = session.ID
	payment.ProviderLiveMode = session.Livemode
	payment.Amount = session.AmountTotal
	payment.Currency = string(session.Currency)
	payment.LastReconciledAt = time.Now()
	if session.Customer != nil {
		payment.CustomerID = session.Customer.ID
	}

	if session.Invoice != nil {
		payment.InvoiceID = session.Invoice.ID
		applyStripeInvoiceToPayment(session.Invoice, payment)
	}
	if session.Subscription != nil {
		payment.SubscriptionID = session.Subscription.ID
		applyStripeSubscriptionToPayment(session.Subscription, payment)
	}
	if session.PaymentIntent != nil {
		applyStripePaymentIntentToPayment(session.PaymentIntent, payment)
	}
	if session.Created > 0 && payment.Created.IsZero() {
		payment.Created = time.Unix(session.Created, 0)
	}
	if stripeSessionPaid(session) && payment.PaidAt.IsZero() {
		if session.Created > 0 {
			payment.PaidAt = time.Unix(session.Created, 0)
		} else {
			payment.PaidAt = time.Now()
		}
	}

	metadata := session.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	if payment.TargetEmail == "" {
		payment.TargetEmail = metadata[stripeMetadataEmail]
	}
	if payment.TargetEmail == "" {
		payment.TargetEmail = session.CustomerEmail
	}
	paymentPlanSnapshotFromMetadata(metadata).apply(payment)
	if payment.Profile == "" {
		payment.Profile = metadata[stripeMetadataProfile]
	}
	if payment.Profile == "" {
		payment.Profile = paymentDefaultProfile
	}
	if payment.InviteCode == "" {
		payment.InviteCode = metadata[stripeMetadataInviteCode]
	}
	if payment.EmailStatus == "" {
		payment.EmailStatus = paymentEmailNotStarted
	}
	if session.Status == stripe.CheckoutSessionStatusExpired {
		setPaymentLifecycleStatus(payment, paymentStatusCheckoutExpired, "Stripe checkout session expired")
	} else if stripeSessionPaid(session) && (payment.Status == "" || payment.Status == paymentStatusCheckoutCreated || payment.Status == paymentStatusCheckoutExpired) {
		payment.Status = paymentStatusPaid
	} else if payment.Status == "" {
		payment.Status = paymentStatusCheckoutCreated
	}
}

func stripeSessionPaid(session *stripe.CheckoutSession) bool {
	return session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid ||
		session.PaymentStatus == stripe.CheckoutSessionPaymentStatusNoPaymentRequired
}

func applyStripeInvoiceToPayment(invoice *stripe.Invoice, payment *Payment) {
	if invoice == nil || invoice.ID == "" {
		return
	}
	payment.InvoiceID = invoice.ID
	payment.ProviderLiveMode = invoice.Livemode
	payment.InvoiceStatus = string(invoice.Status)
	if invoice.Customer != nil {
		payment.CustomerID = invoice.Customer.ID
	}
	if invoice.AmountPaid > 0 {
		payment.Amount = invoice.AmountPaid
	} else if invoice.AmountDue > 0 {
		payment.Amount = invoice.AmountDue
	}
	if invoice.Currency != "" {
		payment.Currency = string(invoice.Currency)
	}
	if invoice.CustomerEmail != "" && payment.TargetEmail == "" {
		payment.TargetEmail = invoice.CustomerEmail
	}
	if invoice.Created > 0 && payment.Created.IsZero() {
		payment.Created = time.Unix(invoice.Created, 0)
	}
	if invoice.Status == stripe.InvoiceStatusPaid && payment.PaidAt.IsZero() {
		if invoice.Created > 0 {
			payment.PaidAt = time.Unix(invoice.Created, 0)
		} else {
			payment.PaidAt = time.Now()
		}
	}
	if subID := stripeInvoiceSubscriptionID(invoice); subID != "" {
		payment.SubscriptionID = subID
	}
	if invoice.Parent != nil && invoice.Parent.SubscriptionDetails != nil {
		metadata := invoice.Parent.SubscriptionDetails.Metadata
		if metadata[stripeMetadataEmail] != "" && payment.TargetEmail == "" {
			payment.TargetEmail = metadata[stripeMetadataEmail]
		}
		paymentPlanSnapshotFromMetadata(metadata).apply(payment)
		if metadata[stripeMetadataProfile] != "" && payment.Profile == "" {
			payment.Profile = metadata[stripeMetadataProfile]
		}
	}

	switch invoice.Status {
	case stripe.InvoiceStatusPaid:
		if payment.Status == "" || payment.Status == paymentStatusCheckoutCreated || payment.Status == paymentStatusFailed || payment.Status == paymentStatusSubscriptionPastDue || payment.Status == paymentStatusSubscriptionLapsed {
			payment.Status = paymentStatusPaid
			payment.Error = ""
		}
	case stripe.InvoiceStatusOpen:
		if invoice.Attempted && invoice.AmountRemaining > 0 {
			setPaymentLifecycleStatus(payment, paymentStatusSubscriptionPastDue, "Stripe invoice payment is past due")
		}
	case stripe.InvoiceStatusUncollectible, stripe.InvoiceStatusVoid:
		setPaymentLifecycleStatus(payment, paymentStatusSubscriptionLapsed, "Stripe invoice is "+string(invoice.Status))
	}
}

func applyStripeSubscriptionToPayment(sub *stripe.Subscription, payment *Payment) {
	if sub == nil || sub.ID == "" {
		return
	}
	payment.SubscriptionID = sub.ID
	payment.ProviderLiveMode = sub.Livemode
	payment.SubscriptionStatus = string(sub.Status)
	if sub.Customer != nil {
		payment.CustomerID = sub.Customer.ID
	}
	payment.SubscriptionCancelAt = sub.CancelAt
	payment.SubscriptionCancelAtPeriodEnd = sub.CancelAtPeriodEnd
	payment.SubscriptionCanceledAt = sub.CanceledAt
	payment.SubscriptionEndedAt = sub.EndedAt
	if sub.Currency != "" && payment.Currency == "" {
		payment.Currency = string(sub.Currency)
	}
	if sub.Created > 0 && payment.Created.IsZero() {
		payment.Created = time.Unix(sub.Created, 0)
	}
	if sub.Metadata != nil {
		if sub.Metadata[stripeMetadataEmail] != "" && payment.TargetEmail == "" {
			payment.TargetEmail = sub.Metadata[stripeMetadataEmail]
		}
		paymentPlanSnapshotFromMetadata(sub.Metadata).apply(payment)
		if sub.Metadata[stripeMetadataProfile] != "" && payment.Profile == "" {
			payment.Profile = sub.Metadata[stripeMetadataProfile]
		}
	}

	switch sub.Status {
	case stripe.SubscriptionStatusCanceled:
		setPaymentLifecycleStatus(payment, paymentStatusSubscriptionCanceled, "")
	case stripe.SubscriptionStatusUnpaid, stripe.SubscriptionStatusIncompleteExpired, stripe.SubscriptionStatusPaused:
		setPaymentLifecycleStatus(payment, paymentStatusSubscriptionLapsed, "Stripe subscription is "+string(sub.Status))
	case stripe.SubscriptionStatusPastDue, stripe.SubscriptionStatusIncomplete:
		setPaymentLifecycleStatus(payment, paymentStatusSubscriptionPastDue, "Stripe subscription is "+string(sub.Status))
	case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusTrialing:
		if sub.CancelAtPeriodEnd {
			setPaymentLifecycleStatus(payment, paymentStatusSubscriptionCanceling, stripeSubscriptionCancelingDetail(sub))
		} else {
			clearPaymentLifecycleStatus(payment)
		}
	}
}

func applyStripePaymentIntentToPayment(paymentIntent *stripe.PaymentIntent, payment *Payment) {
	if paymentIntent == nil || paymentIntent.ID == "" {
		return
	}
	payment.PaymentIntentID = paymentIntent.ID
	payment.ProviderLiveMode = paymentIntent.Livemode
	if paymentIntent.Customer != nil {
		payment.CustomerID = paymentIntent.Customer.ID
	}
	if paymentIntent.AmountReceived > 0 {
		payment.Amount = paymentIntent.AmountReceived
	} else if paymentIntent.Amount > 0 && payment.Amount == 0 {
		payment.Amount = paymentIntent.Amount
	}
	if paymentIntent.Currency != "" {
		payment.Currency = string(paymentIntent.Currency)
	}
	if paymentIntent.ReceiptEmail != "" && payment.TargetEmail == "" {
		payment.TargetEmail = paymentIntent.ReceiptEmail
	}
	if paymentIntent.Created > 0 && payment.Created.IsZero() {
		payment.Created = time.Unix(paymentIntent.Created, 0)
	}
	if paymentIntent.LatestCharge != nil {
		payment.ChargeID = paymentIntent.LatestCharge.ID
		applyStripeChargeToPayment(paymentIntent.LatestCharge, payment)
	}

	switch paymentIntent.Status {
	case stripe.PaymentIntentStatusSucceeded:
		if payment.PaidAt.IsZero() {
			if paymentIntent.Created > 0 {
				payment.PaidAt = time.Unix(paymentIntent.Created, 0)
			} else {
				payment.PaidAt = time.Now()
			}
		}
		if payment.Status == "" || payment.Status == paymentStatusCheckoutCreated || payment.Status == paymentStatusPaymentCanceled || payment.Status == paymentStatusFailed {
			payment.Status = paymentStatusPaid
			payment.Error = ""
		}
	case stripe.PaymentIntentStatusCanceled:
		setPaymentLifecycleStatus(payment, paymentStatusPaymentCanceled, stripePaymentIntentCanceledDetail(paymentIntent))
	case stripe.PaymentIntentStatusRequiresPaymentMethod:
		setPaymentLifecycleStatus(payment, paymentStatusFailed, "Stripe payment requires a new payment method")
	}
}

func applyStripeChargeToPayment(charge *stripe.Charge, payment *Payment) {
	if charge == nil || charge.ID == "" {
		return
	}
	payment.ChargeID = charge.ID
	payment.ProviderLiveMode = charge.Livemode
	if charge.Customer != nil {
		payment.CustomerID = charge.Customer.ID
	}
	if charge.PaymentIntent != nil {
		payment.PaymentIntentID = charge.PaymentIntent.ID
	}
	if charge.Amount > 0 && payment.Amount == 0 {
		payment.Amount = charge.Amount
	}
	if charge.Currency != "" {
		payment.Currency = string(charge.Currency)
	}
	if charge.ReceiptEmail != "" && payment.TargetEmail == "" {
		payment.TargetEmail = charge.ReceiptEmail
	}
	if charge.Created > 0 && payment.Created.IsZero() {
		payment.Created = time.Unix(charge.Created, 0)
	}
	payment.RefundedAmount = charge.AmountRefunded
	if charge.AmountRefunded > 0 {
		if charge.Refunded || charge.AmountRefunded >= charge.Amount {
			setPaymentLifecycleStatus(payment, paymentStatusRefunded, "")
		} else {
			setPaymentLifecycleStatus(payment, paymentStatusPartiallyRefunded, "")
		}
	}
	if charge.Status == stripe.ChargeStatusFailed {
		setPaymentLifecycleStatus(payment, paymentStatusFailed, firstNonEmpty(charge.FailureMessage, charge.FailureCode, "Stripe charge failed"))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func stripeSessionInvoiceID(session *stripe.CheckoutSession) string {
	if session == nil || session.Invoice == nil {
		return ""
	}
	return session.Invoice.ID
}

func stripeSessionSubscriptionID(session *stripe.CheckoutSession) string {
	if session == nil || session.Subscription == nil {
		return ""
	}
	return session.Subscription.ID
}

func stripeSessionPaymentIntentID(session *stripe.CheckoutSession) string {
	if session == nil || session.PaymentIntent == nil {
		return ""
	}
	return session.PaymentIntent.ID
}

func stripeInvoiceSubscriptionID(invoice *stripe.Invoice) string {
	if invoice == nil || invoice.Parent == nil || invoice.Parent.SubscriptionDetails == nil || invoice.Parent.SubscriptionDetails.Subscription == nil {
		return ""
	}
	return invoice.Parent.SubscriptionDetails.Subscription.ID
}

func stripeInvoicePaymentIntentID(invoice *stripe.Invoice) string {
	if invoice == nil || invoice.Payments == nil {
		return ""
	}
	for _, payment := range invoice.Payments.Data {
		if payment.Payment != nil && payment.Payment.PaymentIntent != nil {
			return payment.Payment.PaymentIntent.ID
		}
	}
	return ""
}

func stripePaymentIntentChargeID(paymentIntent *stripe.PaymentIntent) string {
	if paymentIntent == nil || paymentIntent.LatestCharge == nil {
		return ""
	}
	return paymentIntent.LatestCharge.ID
}

func stripePaymentIntentCanceledDetail(paymentIntent *stripe.PaymentIntent) string {
	if paymentIntent == nil || paymentIntent.CancellationReason == "" {
		return "Stripe payment was canceled"
	}
	return "Stripe payment was canceled: " + string(paymentIntent.CancellationReason)
}

func stripeSubscriptionCancelingDetail(sub *stripe.Subscription) string {
	if sub == nil || sub.CancelAt == 0 {
		return "Stripe subscription will cancel at period end"
	}
	return "Stripe subscription will cancel at " + time.Unix(sub.CancelAt, 0).Format(time.RFC3339)
}
