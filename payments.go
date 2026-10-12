package main

import (
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	lm "github.com/hrfee/jfa-go/logmessages"
)

const (
	paymentDefaultProfile = "Default"
	paymentPlanMonthly    = "Monthly"
	paymentPlanStandard   = "Standard"

	paymentStatusCheckoutCreated       = "checkout_created"
	paymentStatusCheckoutExpired       = "checkout_expired"
	paymentStatusPaid                  = "paid"
	paymentStatusFulfilled             = "fulfilled"
	paymentStatusEmailSent             = "email_sent"
	paymentStatusEmailFailed           = "email_failed"
	paymentStatusPaymentCanceled       = "payment_canceled"
	paymentStatusRefunded              = "refunded"
	paymentStatusPartiallyRefunded     = "partially_refunded"
	paymentStatusSubscriptionCanceling = "subscription_canceling"
	paymentStatusSubscriptionCanceled  = "subscription_canceled"
	paymentStatusSubscriptionPastDue   = "subscription_past_due"
	paymentStatusSubscriptionLapsed    = "subscription_lapsed"
	paymentStatusFailed                = "failed"
	paymentStatusNeedsReview           = "needs_review"

	paymentEmailNotStarted    = "not_started"
	paymentEmailNotApplicable = "not_applicable"
	paymentEmailPending       = "pending"
	paymentEmailSent          = "sent"
	paymentEmailFailed        = "failed"
	paymentEmailDisabled      = "disabled"

	paymentInviteExpiredNeedsReview = "Paid invite expired before redemption"
)

type paymentFulfillment struct {
	Provider            string
	TransactionID       string
	SubscriptionID      string
	TargetEmail         string
	PlanID              string
	Plan                string
	Profile             string
	AccessMonths        int
	AccessDays          int
	Recurring           bool
	StripeInterval      string
	StripeIntervalCount int64
}

type paymentFulfillmentResult struct {
	Duplicate        bool
	Invite           Invite
	InviteCode       string
	JellyfinID       string
	Expiry           time.Time
	ShouldSendInvite bool
}

func paymentUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func paymentStatusPriority(status string) int {
	switch status {
	case paymentStatusRefunded:
		return 110
	case paymentStatusSubscriptionCanceled:
		return 100
	case paymentStatusSubscriptionLapsed:
		return 95
	case paymentStatusPartiallyRefunded:
		return 90
	case paymentStatusSubscriptionPastDue:
		return 80
	case paymentStatusSubscriptionCanceling:
		return 70
	case paymentStatusPaymentCanceled, paymentStatusCheckoutExpired:
		return 60
	case paymentStatusNeedsReview:
		return 50
	case paymentStatusEmailFailed, paymentStatusFailed:
		return 40
	case paymentStatusEmailSent, paymentStatusFulfilled:
		return 30
	case paymentStatusPaid:
		return 20
	case paymentStatusCheckoutCreated:
		return 10
	default:
		return 0
	}
}

func paymentStatusIsLifecycle(status string) bool {
	switch status {
	case paymentStatusRefunded,
		paymentStatusPartiallyRefunded,
		paymentStatusSubscriptionCanceling,
		paymentStatusSubscriptionCanceled,
		paymentStatusSubscriptionPastDue,
		paymentStatusSubscriptionLapsed,
		paymentStatusPaymentCanceled,
		paymentStatusCheckoutExpired:
		return true
	default:
		return false
	}
}

func paymentStatusIsRefund(status string) bool {
	return status == paymentStatusRefunded || status == paymentStatusPartiallyRefunded
}

func setPaymentLifecycleStatus(payment *Payment, status, detail string) {
	if status == "" {
		return
	}
	if payment.Status == "" || paymentStatusPriority(status) >= paymentStatusPriority(payment.Status) {
		payment.Status = status
		payment.Error = detail
	}
}

func clearPaymentLifecycleStatus(payment *Payment) {
	if paymentStatusIsRefund(payment.Status) {
		return
	}
	if !paymentStatusIsLifecycle(payment.Status) {
		return
	}
	payment.Error = ""
	if !payment.FulfilledAt.IsZero() {
		if payment.EmailStatus == paymentEmailSent {
			payment.Status = paymentStatusEmailSent
			return
		}
		if payment.EmailStatus == paymentEmailFailed {
			payment.Status = paymentStatusEmailFailed
			return
		}
		payment.Status = paymentStatusFulfilled
		return
	}
	if !payment.PaidAt.IsZero() {
		payment.Status = paymentStatusPaid
		return
	}
	payment.Status = paymentStatusCheckoutCreated
}

func paymentToDTO(payment Payment) paymentDTO {
	return paymentDTO{
		ID:                            payment.ID,
		Provider:                      payment.Provider,
		InstanceID:                    payment.InstanceID,
		ProviderPaymentID:             payment.ProviderPaymentID,
		ProviderLiveMode:              payment.ProviderLiveMode,
		CustomerID:                    payment.CustomerID,
		PaymentIntentID:               payment.PaymentIntentID,
		ChargeID:                      payment.ChargeID,
		InvoiceID:                     payment.InvoiceID,
		SubscriptionID:                payment.SubscriptionID,
		TargetEmail:                   payment.TargetEmail,
		PlanID:                        payment.PlanID,
		Plan:                          payment.Plan,
		Profile:                       payment.Profile,
		AccessMonths:                  payment.AccessMonths,
		AccessDays:                    payment.AccessDays,
		Recurring:                     payment.Recurring,
		StripeInterval:                payment.StripeInterval,
		StripeIntervalCount:           payment.StripeIntervalCount,
		Amount:                        payment.Amount,
		RefundedAmount:                payment.RefundedAmount,
		Currency:                      payment.Currency,
		Status:                        payment.Status,
		EmailStatus:                   payment.EmailStatus,
		InvoiceStatus:                 payment.InvoiceStatus,
		SubscriptionStatus:            payment.SubscriptionStatus,
		SubscriptionCancelAt:          payment.SubscriptionCancelAt,
		SubscriptionCancelAtPeriodEnd: payment.SubscriptionCancelAtPeriodEnd,
		SubscriptionCanceledAt:        payment.SubscriptionCanceledAt,
		SubscriptionEndedAt:           payment.SubscriptionEndedAt,
		SubscriptionCancelNotifiedAt:  paymentUnix(payment.SubscriptionCancelNotifiedAt),
		InviteCode:                    payment.InviteCode,
		JellyfinID:                    payment.JellyfinID,
		Error:                         payment.Error,
		Created:                       paymentUnix(payment.Created),
		Updated:                       paymentUnix(payment.Updated),
		PaidAt:                        paymentUnix(payment.PaidAt),
		FulfilledAt:                   paymentUnix(payment.FulfilledAt),
		EmailSentAt:                   paymentUnix(payment.EmailSentAt),
		LastReconciledAt:              paymentUnix(payment.LastReconciledAt),
	}
}

func (app *appContext) GetPayments(gc *gin.Context) {
	payments := app.storage.GetPayments()
	sort.Slice(payments, func(i, j int) bool {
		return payments[i].Created.After(payments[j].Created)
	})

	dto := getPaymentsDTO{Payments: make([]paymentDTO, len(payments))}
	for i, payment := range payments {
		dto.Payments[i] = paymentToDTO(payment)
	}
	gc.JSON(200, dto)
}

func (app *appContext) ResendPaymentInvite(gc *gin.Context) {
	paymentID := gc.Param("id")
	payment, ok := app.storage.GetPaymentKey(paymentID)
	if !ok {
		respond(404, "Payment not found", gc)
		return
	}
	if payment.InviteCode == "" {
		respond(400, "Payment has no invite to resend", gc)
		return
	}
	if payment.TargetEmail == "" {
		respond(400, "Payment has no target email", gc)
		return
	}
	if !emailEnabled {
		respond(400, "Email is disabled", gc)
		return
	}

	invite, ok := app.storage.GetInvitesKey(payment.InviteCode)
	if !ok {
		respond(404, "Invite not found", gc)
		return
	}
	if !paymentCanRecoverInvite(payment) {
		respond(400, "Payment is not eligible for invite resend", gc)
		return
	}

	invite = app.refreshPurchasedInviteForResend(payment, invite)
	app.sendPurchasedInvite(invite, payment.TargetEmail, paymentID, payment.Plan)
	gc.JSON(200, stringResponse{Response: "Invite queued"})
}

func (app *appContext) setPayment(id string, mutate func(*Payment)) {
	if id == "" {
		return
	}

	payment, ok := app.storage.GetPaymentKey(id)
	now := time.Now()
	if !ok {
		payment = Payment{
			ID:      id,
			Created: now,
		}
	}
	if payment.Created.IsZero() {
		payment.Created = now
	}

	mutate(&payment)

	payment.ID = id
	payment.Updated = now
	app.storage.SetPaymentKey(id, payment)
}

func (app *appContext) setPaymentsByStripeIDs(paymentIntentID, chargeID, invoiceID, subscriptionID string, mutate func(*Payment)) bool {
	seen := map[string]bool{}
	found := false
	for _, payment := range app.storage.GetPayments() {
		if payment.Provider != lm.Stripe {
			continue
		}
		if seen[payment.ID] || !paymentMatchesStripeIDs(payment, paymentIntentID, chargeID, invoiceID, subscriptionID) {
			continue
		}
		seen[payment.ID] = true
		found = true
		app.setPayment(payment.ID, mutate)
	}
	return found
}

func paymentMatchesStripeIDs(payment Payment, paymentIntentID, chargeID, invoiceID, subscriptionID string) bool {
	return stripeIDMatches(paymentIntentID, payment.PaymentIntentID, payment.ProviderPaymentID, payment.ID) ||
		stripeIDMatches(chargeID, payment.ChargeID, payment.ProviderPaymentID, payment.ID) ||
		stripeIDMatches(invoiceID, payment.InvoiceID, payment.ProviderPaymentID, payment.ID) ||
		stripeIDMatches(subscriptionID, payment.SubscriptionID, payment.ProviderPaymentID, payment.ID)
}

func (app *appContext) stripeSubscriptionForUser(userID string) (Payment, bool) {
	if userID == "" {
		return Payment{}, false
	}

	email := ""
	if emailStore, ok := app.storage.GetEmailsKey(userID); ok {
		email = emailStore.Addr
	}

	instanceID := app.paymentInstanceID()
	bestScore := int64(-1)
	best := Payment{}
	for _, payment := range app.storage.GetPayments() {
		if payment.Provider != lm.Stripe || payment.SubscriptionID == "" {
			continue
		}
		if payment.InstanceID != "" && payment.InstanceID != instanceID {
			continue
		}
		if payment.JellyfinID != userID && (email == "" || !strings.EqualFold(payment.TargetEmail, email)) {
			continue
		}

		score := stripeSubscriptionPaymentScore(payment)
		if score > bestScore {
			bestScore = score
			best = payment
		}
	}
	return best, bestScore >= 0
}

func stripeSubscriptionPaymentScore(payment Payment) int64 {
	statusScore := int64(0)
	switch payment.SubscriptionStatus {
	case "active", "trialing":
		statusScore = 500
	case "past_due", "incomplete":
		statusScore = 400
	case "unpaid", "paused", "incomplete_expired":
		statusScore = 200
	case "canceled":
		statusScore = 100
	}

	switch payment.Status {
	case paymentStatusSubscriptionCanceling:
		statusScore += 60
	case paymentStatusPaid, paymentStatusFulfilled, paymentStatusEmailSent:
		statusScore += 40
	case paymentStatusSubscriptionPastDue:
		statusScore += 30
	case paymentStatusSubscriptionCanceled, paymentStatusSubscriptionLapsed, paymentStatusRefunded:
		statusScore -= 50
	}
	if payment.SubscriptionCancelAtPeriodEnd {
		statusScore += 20
	}

	created := payment.Created.Unix()
	if created < 0 {
		created = 0
	}
	return statusScore*1_000_000_000_000 + created
}

func (app *appContext) shouldSkipExpiredPaidUser(userID string, expiry UserExpiry) bool {
	return app.shouldSkipExpiredPaidUserWithReconcile(userID, expiry, app.reconcileStripePayments)
}

func (app *appContext) shouldSkipExpiredPaidUserWithReconcile(userID string, expiry UserExpiry, reconcile func() ReconcilePaymentsDTO) bool {
	if !stripeEnabled {
		return false
	}

	payment, ok := app.stripeSubscriptionForUser(userID)
	if !ok || !stripeSubscriptionGrantsUserAccess(payment, expiry, time.Now()) {
		return false
	}

	if reconcile != nil {
		result := reconcile()
		if result.Error != "" && app.err != nil {
			app.err.Printf("Stripe reconciliation failed while checking paid expiry for %s: %s", userID, result.Error)
		}
	}

	payment, ok = app.stripeSubscriptionForUser(userID)
	return ok && stripeSubscriptionGrantsUserAccess(payment, expiry, time.Now())
}

func stripeSubscriptionGrantsUserAccess(payment Payment, expiry UserExpiry, now time.Time) bool {
	if payment.Provider != lm.Stripe || payment.SubscriptionID == "" {
		return false
	}

	switch payment.Status {
	case paymentStatusRefunded,
		paymentStatusSubscriptionCanceled,
		paymentStatusSubscriptionLapsed,
		paymentStatusPaymentCanceled,
		paymentStatusCheckoutExpired:
		return false
	}

	switch payment.SubscriptionStatus {
	case "active", "trialing":
		if payment.SubscriptionCancelAt > 0 || payment.SubscriptionCancelAtPeriodEnd || payment.Status == paymentStatusSubscriptionCanceling {
			return stripeSubscriptionAccessUntil(payment, expiry).After(now)
		}
		return true
	default:
		if payment.Status == paymentStatusSubscriptionCanceling {
			return stripeSubscriptionAccessUntil(payment, expiry).After(now)
		}
		return false
	}
}

func stripeSubscriptionAccessUntil(payment Payment, expiry UserExpiry) time.Time {
	if payment.SubscriptionCancelAt > 0 {
		return time.Unix(payment.SubscriptionCancelAt, 0)
	}
	if !expiry.Expiry.IsZero() {
		return expiry.Expiry
	}
	return time.Time{}
}

func (app *appContext) mySubscriptionDTO(userID string) *MySubscriptionDTO {
	payment, ok := app.stripeSubscriptionForUser(userID)
	if !ok {
		return nil
	}

	dto := &MySubscriptionDTO{
		Provider:          payment.Provider,
		SubscriptionID:    payment.SubscriptionID,
		Status:            payment.SubscriptionStatus,
		PaymentStatus:     payment.Status,
		CancelAtPeriodEnd: payment.SubscriptionCancelAtPeriodEnd,
		CancelAt:          payment.SubscriptionCancelAt,
		CanceledAt:        payment.SubscriptionCanceledAt,
		EndedAt:           payment.SubscriptionEndedAt,
		Amount:            payment.Amount,
		Currency:          payment.Currency,
	}
	if dto.Status == "" {
		dto.Status = payment.Status
	}
	if expiry, ok := app.storage.GetUserExpiryKey(userID); ok {
		dto.PaidThrough = paymentUnix(expiry.Expiry)
		if dto.CancelAt == 0 && dto.CancelAtPeriodEnd {
			dto.CancelAt = dto.PaidThrough
		}
	}
	return dto
}

func stripeIDMatches(id string, candidates ...string) bool {
	if id == "" {
		return false
	}
	for _, candidate := range candidates {
		if candidate == id {
			return true
		}
	}
	return false
}

func (app *appContext) markPaymentFulfilled(id string, result paymentFulfillmentResult) {
	app.setPayment(id, func(payment *Payment) {
		if result.InviteCode != "" {
			payment.InviteCode = result.InviteCode
		}
		if result.JellyfinID != "" {
			payment.JellyfinID = result.JellyfinID
		}
		if payment.FulfilledAt.IsZero() {
			payment.FulfilledAt = time.Now()
		}
		if payment.EmailStatus == "" {
			payment.EmailStatus = paymentEmailNotStarted
		}
		if payment.Status != paymentStatusEmailSent && payment.Status != paymentStatusEmailFailed && !paymentStatusIsLifecycle(payment.Status) {
			payment.Status = paymentStatusFulfilled
		}
	})
}

func (app *appContext) markPaymentEmail(id, emailStatus, errText string) {
	app.setPayment(id, func(payment *Payment) {
		payment.EmailStatus = emailStatus
		if !paymentStatusIsLifecycle(payment.Status) {
			payment.Error = errText
		}

		switch emailStatus {
		case paymentEmailSent:
			if !paymentStatusIsLifecycle(payment.Status) {
				payment.Status = paymentStatusEmailSent
			}
			payment.EmailSentAt = time.Now()
		case paymentEmailFailed:
			if !paymentStatusIsLifecycle(payment.Status) {
				payment.Status = paymentStatusEmailFailed
			}
		case paymentEmailPending, paymentEmailDisabled:
			if !paymentStatusIsLifecycle(payment.Status) && (payment.Status == "" || payment.Status == paymentStatusPaid || payment.Status == paymentStatusCheckoutCreated) {
				payment.Status = paymentStatusFulfilled
			}
		}
	})
}

func shouldSendStorePaymentConfirmation(payment Payment) bool {
	if payment.JellyfinID == "" || payment.TargetEmail == "" || payment.InviteCode != "" {
		return false
	}
	if paymentStatusIsLifecycle(payment.Status) {
		return false
	}
	switch payment.EmailStatus {
	case paymentEmailSent, paymentEmailPending, paymentEmailDisabled, paymentEmailNotApplicable:
		return false
	default:
		return true
	}
}

func (app *appContext) repairStaleStorePaymentAsInvite(payment Payment) bool {
	if payment.JellyfinID == "" || payment.TargetEmail == "" || payment.InviteCode != "" || paymentStatusIsLifecycle(payment.Status) {
		return false
	}
	if app.jf.MediaBrowser == nil {
		return false
	}
	if _, err := app.jf.UserByID(payment.JellyfinID, false); err == nil {
		return false
	}

	app.info.Printf("Repairing Stripe payment %s for %q: stored Jellyfin user %q no longer exists", payment.ID, payment.TargetEmail, payment.JellyfinID)
	app.storage.DeleteEmailsKey(payment.JellyfinID)
	app.storage.DeleteUserExpiryKey(payment.JellyfinID)

	invite := app.createPurchasedInvite(payment.Provider, payment.TargetEmail, payment.Plan, payment.Profile, payment.ID, payment.AccessMonths, payment.AccessDays)
	app.setPayment(payment.ID, func(payment *Payment) {
		payment.JellyfinID = ""
		payment.InviteCode = invite.Code
		payment.Error = ""
	})
	if emailEnabled {
		app.sendPurchasedInvite(invite, payment.TargetEmail, payment.ID, payment.Plan)
	} else {
		app.markPaymentEmail(payment.ID, paymentEmailDisabled, "")
	}
	return true
}

func (app *appContext) markPaymentError(id, errText string) {
	app.setPayment(id, func(payment *Payment) {
		setPaymentLifecycleStatus(payment, paymentStatusFailed, errText)
	})
}

func (app *appContext) markPaymentSubscriptionCanceled(subscriptionID, targetEmail string) {
	if subscriptionID == "" {
		return
	}

	found := false
	for _, payment := range app.storage.GetPayments() {
		if payment.SubscriptionID != subscriptionID {
			continue
		}
		found = true
		app.setPayment(payment.ID, func(payment *Payment) {
			setPaymentLifecycleStatus(payment, paymentStatusSubscriptionCanceled, "")
		})
	}
	if found {
		return
	}

	app.setPayment("subscription-"+subscriptionID, func(payment *Payment) {
		payment.Provider = lm.Stripe
		payment.InstanceID = app.paymentInstanceID()
		payment.ProviderPaymentID = subscriptionID
		payment.SubscriptionID = subscriptionID
		payment.TargetEmail = targetEmail
		payment.Plan = paymentPlanMonthly
		setPaymentLifecycleStatus(payment, paymentStatusSubscriptionCanceled, "")
		payment.EmailStatus = paymentEmailNotApplicable
	})
}

// findUserByEmail looks up a Jellyfin user ID and their stored EmailAddress by email.
// Returns ("", EmailAddress{}, false) if no match is found.
func (app *appContext) findUserByEmail(addr string) (string, EmailAddress, bool) {
	for _, em := range app.storage.GetEmails() {
		if strings.EqualFold(em.Addr, addr) {
			return em.JellyfinID, em, true
		}
	}
	return "", EmailAddress{}, false
}

func normalizePaymentPlan(plan string) string {
	plan = strings.TrimSpace(plan)
	if strings.EqualFold(plan, paymentPlanMonthly) {
		return paymentPlanMonthly
	}
	if strings.EqualFold(plan, paymentPlanStandard) || plan == "" {
		return paymentPlanStandard
	}
	return plan
}

func paidPlanExpiry(plan string, accessMonths, accessDays int, base time.Time) (time.Time, bool) {
	return paymentPlanExpiry(plan, accessMonths, accessDays, base)
}

func (app *appContext) userHasActivePaidExpiry(addr string) (string, bool) {
	userID, _, found := app.findUserByEmail(addr)
	if !found {
		return "", false
	}

	user, err := app.jf.UserByID(userID, false)
	if err != nil || user.Policy.IsDisabled {
		return userID, false
	}

	expiry, ok := app.storage.GetUserExpiryKey(userID)
	return userID, ok && expiry.Expiry.After(time.Now())
}

func (app *appContext) fulfillStorePayment(f paymentFulfillment) paymentFulfillmentResult {
	f.Plan = normalizePaymentPlan(f.Plan)
	if f.Profile == "" {
		f.Profile = paymentDefaultProfile
	}

	existingUserID, existingEmail, found := app.findUserByEmail(f.TargetEmail)
	if found {
		if app.jf.MediaBrowser != nil {
			if _, err := app.jf.UserByID(existingUserID, false); err != nil {
				app.info.Printf("Ignoring stale email mapping for %q to missing Jellyfin user %q: %v", f.TargetEmail, existingUserID, err)
				app.storage.DeleteEmailsKey(existingUserID)
				app.storage.DeleteUserExpiryKey(existingUserID)
				found = false
			}
		}
	}
	if found {
		app.info.Printf(lm.ExistingUserFound, f.TargetEmail, existingUserID)
		if f.SubscriptionID != "" {
			existingEmail.Label = f.Provider + " Subscription: " + f.SubscriptionID
		} else {
			existingEmail.Label = f.Provider + " Payment: " + f.TransactionID
		}
		app.storage.SetEmailsKey(existingUserID, existingEmail)

		newExpiry, changed := app.extendPaidUserExpiry(existingUserID, f.TransactionID, f.Plan, f.AccessMonths, f.AccessDays)
		if !changed {
			app.info.Printf(lm.PaymentTransactionAlreadyProcessed, f.Provider, f.TransactionID, existingUserID)
			return paymentFulfillmentResult{Duplicate: true, JellyfinID: existingUserID, Expiry: newExpiry}
		}
		app.reEnablePaidUser(existingUserID)
		app.info.Printf(lm.UserReactivated, existingUserID, newExpiry)
		return paymentFulfillmentResult{JellyfinID: existingUserID, Expiry: newExpiry}
	}

	if f.TransactionID != "" {
		if invite, ok := app.purchasedInviteByTransaction(f.TransactionID); ok {
			app.info.Printf(lm.PaymentTransactionAlreadyProcessed, f.Provider, f.TransactionID, "pending invite")
			return paymentFulfillmentResult{Duplicate: true, Invite: invite, InviteCode: invite.Code}
		}
	}

	invite := app.createPurchasedInvite(f.Provider, f.TargetEmail, f.Plan, f.Profile, f.TransactionID, f.AccessMonths, f.AccessDays)
	return paymentFulfillmentResult{
		Invite:           invite,
		InviteCode:       invite.Code,
		ShouldSendInvite: emailEnabled,
	}
}

func (app *appContext) purchasedInviteByTransaction(transactionID string) (Invite, bool) {
	for _, invite := range app.storage.GetInvites() {
		if invite.PaymentID == transactionID {
			return invite, true
		}
	}
	return Invite{}, false
}

func (app *appContext) extendPaidUserExpiry(userID, transactionID, plan string, accessMonths, accessDays int) (time.Time, bool) {
	userExpiry, ok := app.storage.GetUserExpiryKey(userID)
	if !ok {
		userExpiry = UserExpiry{Expiry: time.Now()}
	}
	if transactionID != "" && userExpiry.LastTransactionID == transactionID {
		return userExpiry.Expiry, false
	}

	base := userExpiry.Expiry
	if now := time.Now(); base.Before(now) {
		base = now
	}
	userExpiry.Expiry, _ = paidPlanExpiry(plan, accessMonths, accessDays, base)
	userExpiry.LastTransactionID = transactionID
	app.storage.SetUserExpiryKey(userID, userExpiry)
	return userExpiry.Expiry, true
}

func (app *appContext) reEnablePaidUser(userID string) {
	paramsUser, err := app.jf.UserByID(userID, false)
	if err != nil {
		return
	}
	if err, _, _ = app.SetUserDisabled(paramsUser, false); err != nil {
		app.err.Printf(lm.FailedReEnableUser, userID, err)
		return
	}
	app.InvalidateUserCaches()
}

func (app *appContext) expirePaidUserNow(userID string) {
	userExpiry, _ := app.storage.GetUserExpiryKey(userID)
	userExpiry.Expiry = time.Now().Add(-1 * time.Second)
	app.storage.SetUserExpiryKey(userID, userExpiry)
}

func (app *appContext) createPurchasedInvite(provider, targetEmail, plan, profile, transactionID string, accessMonths, accessDays int) Invite {
	if _, ok := app.storage.GetProfileKey(profile); !ok {
		app.debug.Printf(lm.FailedGetProfile, profile)
		if _, ok := app.storage.GetProfileKey(paymentDefaultProfile); ok {
			profile = paymentDefaultProfile
		} else {
			profile = ""
		}
	}

	inviteCode := GenerateInviteCode()
	invite := Invite{
		Code:          inviteCode,
		Created:       time.Now(),
		Label:         provider + " " + plan + " by " + targetEmail,
		UserLabel:     "Purchased via " + provider,
		RemainingUses: 1,
		Profile:       profile,
		SendTo:        targetEmail,
		PaymentID:     transactionID,
	}

	var userExpiry bool
	invite.ValidTill, userExpiry = paidPlanExpiry(plan, accessMonths, accessDays, time.Now())
	invite.UserExpiry = userExpiry
	if userExpiry {
		invite.UserMonths = accessMonths
		invite.UserDays = accessDays
		if invite.UserMonths == 0 && invite.UserDays == 0 && normalizePaymentPlan(plan) == paymentPlanMonthly {
			invite.UserMonths = 1
		}
	}

	app.storage.SetInvitesKey(inviteCode, invite)
	app.info.Printf(lm.GeneratedInviteForPurchase, inviteCode, targetEmail)
	return invite
}

func (app *appContext) preserveExpiredPaidInvite(invite Invite) bool {
	payment, ok := app.paymentForInvite(invite)
	if !ok && invite.PaymentStatus != paymentStatusPaid {
		return false
	}
	if ok && !paymentCanRecoverInvite(payment) {
		return false
	}

	paymentID := invite.PaymentID
	if ok {
		paymentID = payment.ID
		app.setPayment(payment.ID, func(payment *Payment) {
			if payment.InviteCode == "" {
				payment.InviteCode = invite.Code
			}
			if !paymentStatusIsLifecycle(payment.Status) {
				payment.Status = paymentStatusNeedsReview
				payment.Error = paymentInviteExpiredNeedsReview
			} else if payment.Error == "" {
				payment.Error = paymentInviteExpiredNeedsReview
			}
		})
	}

	app.info.Printf("Preserving expired paid invite %s for payment %s", invite.Code, paymentID)
	return true
}

func (app *appContext) paymentForInvite(invite Invite) (Payment, bool) {
	if invite.PaymentID != "" {
		if payment, ok := app.storage.GetPaymentKey(invite.PaymentID); ok {
			return payment, true
		}
	}
	for _, payment := range app.storage.GetPayments() {
		if payment.InviteCode == invite.Code {
			return payment, true
		}
	}
	return Payment{}, false
}

func paymentCanRecoverInvite(payment Payment) bool {
	switch payment.Status {
	case paymentStatusRefunded,
		paymentStatusSubscriptionCanceled,
		paymentStatusSubscriptionLapsed,
		paymentStatusPaymentCanceled,
		paymentStatusCheckoutExpired:
		return false
	}
	return payment.Status == paymentStatusPaid ||
		payment.Status == paymentStatusFulfilled ||
		payment.Status == paymentStatusEmailSent ||
		payment.Status == paymentStatusEmailFailed ||
		payment.Status == paymentStatusNeedsReview ||
		payment.Status == paymentStatusSubscriptionCanceling ||
		payment.Status == paymentStatusSubscriptionPastDue ||
		!payment.PaidAt.IsZero()
}

func (app *appContext) refreshPurchasedInviteForResend(payment Payment, invite Invite) Invite {
	if !time.Now().After(invite.ValidTill) {
		return invite
	}

	var userExpiry bool
	invite.ValidTill, userExpiry = paidPlanExpiry(payment.Plan, payment.AccessMonths, payment.AccessDays, time.Now())
	if userExpiry {
		invite.UserExpiry = true
		if payment.AccessMonths > 0 || payment.AccessDays > 0 {
			invite.UserMonths = payment.AccessMonths
			invite.UserDays = payment.AccessDays
		} else if invite.UserMonths == 0 && invite.UserDays == 0 && normalizePaymentPlan(payment.Plan) == paymentPlanMonthly {
			invite.UserMonths = 1
		}
	}

	app.storage.SetInvitesKey(invite.Code, invite)
	return invite
}

func (app *appContext) sendPurchasedInvite(invite Invite, targetEmail, paymentID, plan string) {
	if paymentID != "" {
		app.markPaymentEmail(paymentID, paymentEmailPending, "")
	}

	go func() {
		msg, err := app.email.constructPurchasedInvite(&invite, plan, false)
		if err != nil {
			app.err.Printf(lm.FailedConstructInviteMessage, targetEmail, err)
			if paymentID != "" {
				app.markPaymentEmail(paymentID, paymentEmailFailed, err.Error())
			}
			return
		}
		if err = app.email.send(msg, targetEmail); err != nil {
			app.err.Printf(lm.FailedSendInviteMessage, invite.Code, targetEmail, err)
			if paymentID != "" {
				app.markPaymentEmail(paymentID, paymentEmailFailed, err.Error())
			}
		} else {
			app.info.Printf(lm.SentInviteMessage, invite.Code, targetEmail)
			if paymentID != "" {
				app.markPaymentEmail(paymentID, paymentEmailSent, "")
			}
		}
	}()
}

func (app *appContext) sendStorePaymentConfirmation(userID, targetEmail, paymentID, provider, plan string, expiry time.Time, recurring bool) {
	if paymentID == "" || targetEmail == "" {
		return
	}
	if !emailEnabled {
		app.markPaymentEmail(paymentID, paymentEmailDisabled, "")
		return
	}
	if app.email == nil || app.email.sender == nil {
		app.markPaymentEmail(paymentID, paymentEmailFailed, "email sender is not configured")
		return
	}

	app.markPaymentEmail(paymentID, paymentEmailPending, "")
	go func() {
		username := targetEmail
		if userID != "" && app.jf.MediaBrowser != nil {
			if user, err := app.jf.UserByID(userID, false); err == nil && user.Name != "" {
				username = user.Name
			}
		}

		msg := app.constructStorePaymentConfirmationMessage(username, provider, plan, expiry, recurring)
		if err := app.email.send(msg, targetEmail); err != nil {
			app.err.Printf("Failed to send %s payment confirmation email for %s to %s: %v", provider, paymentID, targetEmail, err)
			app.markPaymentEmail(paymentID, paymentEmailFailed, err.Error())
			return
		}
		app.info.Printf("Sent %s payment confirmation email for %s to %s", provider, paymentID, targetEmail)
		app.markPaymentEmail(paymentID, paymentEmailSent, "")
	}()
}

func (app *appContext) constructStorePaymentConfirmationMessage(username, provider, plan string, expiry time.Time, recurring bool) *Message {
	serverName := serverHeader(app.config, nil)
	if plan == "" {
		plan = "access"
	}
	if provider == "" {
		provider = "payment"
	}

	lines := []string{
		"Hi " + username + ",",
		"",
		"Your " + serverName + " subscription is active.",
		"Plan: " + plan,
	}
	if !expiry.IsZero() {
		lines = append(lines, "Paid through: "+formatDatetime(expiry))
	}
	if recurring {
		lines = append(lines, "Future renewals will be handled by "+provider+".")
	}
	if accountURL := strings.TrimRight(ExternalURI(nil), "/") + PAGES.MyAccount; accountURL != "" && PAGES.MyAccount != "" && PAGES.MyAccount != "disabled" {
		lines = append(lines, "", "Manage your account: "+accountURL)
	}
	lines = append(lines, "", "If you did not make this purchase, contact the administrator.")

	return &Message{
		Subject: "Subscription active - " + serverName,
		Text:    strings.Join(lines, "\n"),
	}
}
