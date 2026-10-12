package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/logger"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/stripe/stripe-go/v86"
	"github.com/timshannon/badgerhold/v4"
	"gopkg.in/ini.v1"
)

func signedStripeHeader(payload []byte, secret string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", ts, payload)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func stripeTestPayload(apiVersion string) []byte {
	return []byte(fmt.Sprintf(`{
		"id": "evt_test_webhook_compat",
		"object": "event",
		"api_version": %q,
		"created": 1782620000,
		"type": "ping",
		"livemode": false,
		"data": {"object": {"object": "ping"}}
	}`, apiVersion))
}

func TestStripeWebhookSupportedAPIVersion(t *testing.T) {
	secret := "whsec_test"
	payload := stripeTestPayload(supportedStripeWebhookAPIVersion)
	header := signedStripeHeader(payload, secret, time.Now().Unix())

	event, err := HandleWebhook(payload, header, secret, true)
	if err != nil {
		t.Fatalf("supported Stripe webhook API version %q was rejected by stripe-go %s (%s): %v", supportedStripeWebhookAPIVersion, stripe.ClientVersion, stripe.APIVersion, err)
	}
	if event.APIVersion != supportedStripeWebhookAPIVersion {
		t.Fatalf("expected API version %q, got %q", supportedStripeWebhookAPIVersion, event.APIVersion)
	}
}

func TestStripeWebhookRejectsUnsupportedReleaseTrain(t *testing.T) {
	secret := "whsec_test"
	payload := stripeTestPayload("2099-01-01.future")
	header := signedStripeHeader(payload, secret, time.Now().Unix())

	_, err := HandleWebhook(payload, header, secret, true)
	if err == nil {
		t.Fatal("expected unsupported Stripe webhook API release train to be rejected")
	}
	if !strings.Contains(err.Error(), "API version") {
		t.Fatalf("expected API version error, got: %v", err)
	}
}

func newStripePaymentTestApp(t *testing.T) *appContext {
	t.Helper()
	opts := badgerhold.DefaultOptions
	opts.Dir = t.TempDir()
	opts.ValueDir = opts.Dir
	opts.Logger = nil

	db, err := badgerhold.Open(opts)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	conf, err := ini.Load([]byte("[stripe]\ninstance_id = instance_test\n"))
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	oldEmailEnabled := emailEnabled
	emailEnabled = false
	t.Cleanup(func() {
		emailEnabled = oldEmailEnabled
	})

	storage := &Storage{db: db, debug: logger.NewEmptyLogger(), logActions: func(string) DebugLogAction { return NoLog }}
	storage.SetProfileKey(paymentDefaultProfile, Profile{Default: true})

	return &appContext{
		config:  &Config{File: conf},
		storage: storage,
		LoggerSet: LoggerSet{
			info:  logger.NewEmptyLogger(),
			debug: logger.NewEmptyLogger(),
			err:   logger.NewEmptyLogger(),
		},
	}
}

func stripeReconcileSession(instanceID string) *stripe.CheckoutSession {
	return &stripe.CheckoutSession{
		ID:            "cs_test_reconcile",
		AmountTotal:   200,
		Currency:      stripe.CurrencyUSD,
		Created:       1782620000,
		PaymentStatus: stripe.CheckoutSessionPaymentStatusPaid,
		Status:        stripe.CheckoutSessionStatusComplete,
		Metadata: map[string]string{
			stripeMetadataSource:     stripeMetadataSourceJFA,
			stripeMetadataInstanceID: instanceID,
			stripeMetadataEmail:      "test@example.com",
			stripeMetadataPlan:       paymentPlanMonthly,
			stripeMetadataProfile:    paymentDefaultProfile,
		},
	}
}

func TestStripePaymentMetadataIncludesAndPersistsInstanceID(t *testing.T) {
	confPath := filepath.Join(t.TempDir(), "config.ini")
	if err := os.WriteFile(confPath, []byte("[stripe]\n"), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	conf, err := ini.Load(confPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	app := &appContext{
		config:     &Config{File: conf},
		configPath: confPath,
	}

	metadata := app.stripePaymentMetadata(map[string]string{
		stripeMetadataEmail:   "test@example.com",
		stripeMetadataPlan:    paymentPlanMonthly,
		stripeMetadataProfile: paymentDefaultProfile,
	})

	if metadata[stripeMetadataSource] != stripeMetadataSourceJFA {
		t.Fatalf("expected metadata source %q, got %q", stripeMetadataSourceJFA, metadata[stripeMetadataSource])
	}
	instanceID := metadata[stripeMetadataInstanceID]
	if !strings.HasPrefix(instanceID, "jfa_") {
		t.Fatalf("expected generated instance ID to start with jfa_, got %q", instanceID)
	}

	reloaded, err := ini.Load(confPath)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if got := reloaded.Section("stripe").Key("instance_id").String(); got != instanceID {
		t.Fatalf("expected persisted instance ID %q, got %q", instanceID, got)
	}
}

func TestPaidInvitePaymentLockRequiresHashedToken(t *testing.T) {
	app := newStripePaymentTestApp(t)
	token, hash, err := newPaymentLockToken()
	if err != nil {
		t.Fatalf("failed to create payment lock: %v", err)
	}

	app.storage.SetPaymentKey("cs_lock", Payment{
		InviteCode:          "invite_lock",
		InviteLockHash:      hash,
		InviteLockCreatedAt: time.Now(),
	})
	invite := Invite{
		Code:            "invite_lock",
		RequiredPayment: true,
		PaymentID:       "cs_lock",
		PaymentStatus:   paymentStatusPaid,
	}

	if !app.validPaidInvitePaymentLock(invite, token) {
		t.Fatal("expected matching payment lock token to be valid")
	}
	if app.validPaidInvitePaymentLock(invite, invite.Code) {
		t.Fatal("did not expect invite code cookie to unlock a hashed payment lock")
	}
	if app.validPaidInvitePaymentLock(invite, "") {
		t.Fatal("did not expect an empty payment lock token to be valid")
	}
}

func TestPaidInvitePaymentLockAllowsLegacyCodeCookieOnlyWithoutHash(t *testing.T) {
	app := newStripePaymentTestApp(t)
	invite := Invite{
		Code:            "invite_legacy",
		RequiredPayment: true,
		PaymentStatus:   paymentStatusPaid,
	}

	if !app.validPaidInvitePaymentLock(invite, invite.Code) {
		t.Fatal("expected legacy invite code cookie to be valid when no hashed lock exists")
	}
	if app.validPaidInvitePaymentLock(invite, "wrong") {
		t.Fatal("did not expect mismatched legacy payment lock cookie to be valid")
	}
}

func TestStripeReconcileRecoversPaidStoreCheckout(t *testing.T) {
	app := newStripePaymentTestApp(t)
	summary := app.reconcileStripeCheckoutSessions("instance_test", []*stripe.CheckoutSession{
		stripeReconcileSession("instance_test"),
	})

	if summary.Scanned != 1 || summary.Matched != 1 || summary.Created != 1 || summary.NeedsReview != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	payment, ok := app.storage.GetPaymentKey("cs_test_reconcile")
	if !ok {
		t.Fatal("expected reconciled payment to be stored")
	}
	if payment.Status != paymentStatusFulfilled {
		t.Fatalf("expected status %q, got %q", paymentStatusFulfilled, payment.Status)
	}
	if payment.InstanceID != "instance_test" {
		t.Fatalf("expected instance ID recorded, got %q", payment.InstanceID)
	}
	if payment.TargetEmail != "test@example.com" || payment.Plan != paymentPlanMonthly || payment.Amount != 200 || payment.Currency != string(stripe.CurrencyUSD) {
		t.Fatalf("unexpected payment fields: %+v", payment)
	}
	if payment.InviteCode == "" {
		t.Fatalf("expected reconciled payment to create a purchased invite: %+v", payment)
	}
	if payment.EmailStatus != paymentEmailDisabled {
		t.Fatalf("expected email status %q, got %q", paymentEmailDisabled, payment.EmailStatus)
	}
}

func TestStripeReconcileIgnoresDifferentInstance(t *testing.T) {
	app := newStripePaymentTestApp(t)
	summary := app.reconcileStripeCheckoutSessions("instance_test", []*stripe.CheckoutSession{
		stripeReconcileSession("other_instance"),
	})

	if summary.Scanned != 1 || summary.Skipped != 1 || summary.Matched != 0 || summary.Created != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if payments := app.storage.GetPayments(); len(payments) != 0 {
		t.Fatalf("expected no payments, got %d", len(payments))
	}
}

func TestStripeReconcileRecoversExistingInviteLink(t *testing.T) {
	app := newStripePaymentTestApp(t)
	app.storage.SetInvitesKey("invite_test", Invite{Code: "invite_test"})
	session := stripeReconcileSession("instance_test")
	session.Metadata[stripeMetadataInviteCode] = "invite_test"

	summary := app.reconcileStripeCheckoutSessions("instance_test", []*stripe.CheckoutSession{session})
	if summary.Created != 1 || summary.NeedsReview != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	payment, ok := app.storage.GetPaymentKey("cs_test_reconcile")
	if !ok {
		t.Fatal("expected reconciled payment to be stored")
	}
	if payment.Status == paymentStatusNeedsReview {
		t.Fatalf("did not expect needs_review for existing invite link: %+v", payment)
	}
	if payment.InviteCode != "invite_test" {
		t.Fatalf("expected invite link to be recovered, got %q", payment.InviteCode)
	}
	invite, ok := app.storage.GetInvitesKey("invite_test")
	if !ok {
		t.Fatal("expected invite link to remain stored")
	}
	if invite.PaymentID != "cs_test_reconcile" || invite.PaymentStatus != paymentStatusPaid {
		t.Fatalf("expected invite payment link to be recorded, got %+v", invite)
	}
}

func TestStripeCheckoutExpiredMarksLifecycleStatus(t *testing.T) {
	session := stripeReconcileSession("instance_test")
	session.PaymentStatus = stripe.CheckoutSessionPaymentStatusUnpaid
	session.Status = stripe.CheckoutSessionStatusExpired

	payment := Payment{Status: paymentStatusCheckoutCreated}
	applyStripeSessionToPayment("instance_test", session, &payment)

	if payment.Status != paymentStatusCheckoutExpired {
		t.Fatalf("expected status %q, got %q", paymentStatusCheckoutExpired, payment.Status)
	}
	if payment.Error == "" {
		t.Fatal("expected checkout expiration detail to be stored")
	}
}

func TestStripeRefundOverridesEmailFailure(t *testing.T) {
	payment := Payment{
		Amount:      200,
		Currency:    "usd",
		Status:      paymentStatusEmailFailed,
		EmailStatus: paymentEmailFailed,
		Error:       "smtp unavailable",
	}

	applyStripeChargeToPayment(&stripe.Charge{
		ID:             "ch_refunded",
		Amount:         200,
		AmountRefunded: 200,
		Refunded:       true,
		Currency:       stripe.CurrencyUSD,
		Status:         stripe.ChargeStatusSucceeded,
		PaymentIntent:  &stripe.PaymentIntent{ID: "pi_refunded"},
	}, &payment)

	if payment.Status != paymentStatusRefunded {
		t.Fatalf("expected status %q, got %q", paymentStatusRefunded, payment.Status)
	}
	if payment.RefundedAmount != 200 {
		t.Fatalf("expected refunded amount 200, got %d", payment.RefundedAmount)
	}
	if payment.PaymentIntentID != "pi_refunded" || payment.ChargeID != "ch_refunded" {
		t.Fatalf("expected Stripe IDs to be stored, got %+v", payment)
	}
}

func TestStorePaymentConfirmationEligibility(t *testing.T) {
	base := Payment{
		ID:          "cs_existing",
		JellyfinID:  "jf_user",
		TargetEmail: "test@example.com",
		Status:      paymentStatusFulfilled,
		EmailStatus: paymentEmailNotStarted,
	}

	if !shouldSendStorePaymentConfirmation(base) {
		t.Fatal("expected fulfilled existing-user payment to be eligible for confirmation")
	}

	for name, mutate := range map[string]func(*Payment){
		"invite payment": func(p *Payment) { p.InviteCode = "invite_test" },
		"missing user": func(p *Payment) {
			p.JellyfinID = ""
		},
		"already sent": func(p *Payment) {
			p.EmailStatus = paymentEmailSent
		},
		"pending": func(p *Payment) {
			p.EmailStatus = paymentEmailPending
		},
		"canceled": func(p *Payment) {
			p.Status = paymentStatusSubscriptionCanceled
		},
	} {
		t.Run(name, func(t *testing.T) {
			payment := base
			mutate(&payment)
			if shouldSendStorePaymentConfirmation(payment) {
				t.Fatalf("did not expect %s to be eligible", name)
			}
		})
	}
}

func TestStripePartialRefundIsVisible(t *testing.T) {
	payment := Payment{
		Amount: 200,
		Status: paymentStatusFulfilled,
	}

	applyStripeChargeToPayment(&stripe.Charge{
		ID:             "ch_partial",
		Amount:         200,
		AmountRefunded: 75,
		Currency:       stripe.CurrencyUSD,
		Status:         stripe.ChargeStatusSucceeded,
	}, &payment)

	if payment.Status != paymentStatusPartiallyRefunded {
		t.Fatalf("expected status %q, got %q", paymentStatusPartiallyRefunded, payment.Status)
	}
	if payment.RefundedAmount != 75 {
		t.Fatalf("expected refunded amount 75, got %d", payment.RefundedAmount)
	}
}

func TestStripeSubscriptionLapseAndRecoveryAreVisible(t *testing.T) {
	fulfilledAt := time.Now().Add(-time.Hour)
	payment := Payment{
		Status:      paymentStatusFulfilled,
		FulfilledAt: fulfilledAt,
	}

	applyStripeSubscriptionToPayment(&stripe.Subscription{
		ID:     "sub_lapsed",
		Status: stripe.SubscriptionStatusPastDue,
	}, &payment)

	if payment.Status != paymentStatusSubscriptionPastDue {
		t.Fatalf("expected status %q, got %q", paymentStatusSubscriptionPastDue, payment.Status)
	}
	if payment.SubscriptionStatus != string(stripe.SubscriptionStatusPastDue) {
		t.Fatalf("expected provider subscription status recorded, got %q", payment.SubscriptionStatus)
	}

	applyStripeSubscriptionToPayment(&stripe.Subscription{
		ID:     "sub_lapsed",
		Status: stripe.SubscriptionStatusActive,
	}, &payment)

	if payment.Status != paymentStatusFulfilled {
		t.Fatalf("expected active subscription to restore fulfilled status, got %q", payment.Status)
	}
	if !payment.FulfilledAt.Equal(fulfilledAt) {
		t.Fatalf("expected fulfilled timestamp to be preserved")
	}
}

func TestStripeSubscriptionCancelingIsVisibleBeforePeriodEnd(t *testing.T) {
	cancelAt := time.Now().AddDate(0, 0, 7).Unix()
	payment := Payment{Status: paymentStatusFulfilled}

	applyStripeSubscriptionToPayment(&stripe.Subscription{
		ID:                "sub_canceling",
		Status:            stripe.SubscriptionStatusActive,
		CancelAtPeriodEnd: true,
		CancelAt:          cancelAt,
	}, &payment)

	if payment.Status != paymentStatusSubscriptionCanceling {
		t.Fatalf("expected status %q, got %q", paymentStatusSubscriptionCanceling, payment.Status)
	}
	if !strings.Contains(payment.Error, time.Unix(cancelAt, 0).Format("2006-01-02")) {
		t.Fatalf("expected cancel date in detail, got %q", payment.Error)
	}
}

func TestTaskListIncludesStripeWhenEnabled(t *testing.T) {
	oldStripeEnabled := stripeEnabled
	stripeEnabled = true
	t.Cleanup(func() {
		stripeEnabled = oldStripeEnabled
	})

	gin.SetMode(gin.TestMode)
	app := newStripePaymentTestApp(t)
	w := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(w)

	app.TaskList(gc)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := TasksDTO{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode tasks response: %v", err)
	}
	for _, task := range resp.Tasks {
		if task.URL == "/tasks/stripe" {
			return
		}
	}
	t.Fatalf("expected Stripe task in response: %+v", resp.Tasks)
}

func TestExpiredPaidUserSkipsCleanupForActiveStripeSubscription(t *testing.T) {
	oldStripeEnabled := stripeEnabled
	stripeEnabled = true
	t.Cleanup(func() {
		stripeEnabled = oldStripeEnabled
	})

	app := newStripePaymentTestApp(t)
	app.storage.SetEmailsKey("jf_user", EmailAddress{Addr: "test@example.com"})
	app.storage.SetPaymentKey("cs_active", Payment{
		Provider:           lm.Stripe,
		InstanceID:         "instance_test",
		SubscriptionID:     "sub_active",
		TargetEmail:        "test@example.com",
		JellyfinID:         "jf_user",
		Status:             paymentStatusFulfilled,
		SubscriptionStatus: string(stripe.SubscriptionStatusActive),
	})

	reconciled := false
	skip := app.shouldSkipExpiredPaidUserWithReconcile("jf_user", UserExpiry{
		Expiry:            time.Now().Add(-time.Hour),
		DeleteAfterPeriod: true,
	}, func() ReconcilePaymentsDTO {
		reconciled = true
		return ReconcilePaymentsDTO{}
	})

	if !skip {
		t.Fatal("expected active Stripe subscription to skip expiry cleanup")
	}
	if !reconciled {
		t.Fatal("expected active Stripe subscription check to trigger reconciliation")
	}
}

func TestExpiredPaidUserDoesNotSkipCleanupForTerminalStripeSubscription(t *testing.T) {
	oldStripeEnabled := stripeEnabled
	stripeEnabled = true
	t.Cleanup(func() {
		stripeEnabled = oldStripeEnabled
	})

	app := newStripePaymentTestApp(t)
	app.storage.SetEmailsKey("jf_user", EmailAddress{Addr: "test@example.com"})
	app.storage.SetPaymentKey("cs_canceled", Payment{
		Provider:           lm.Stripe,
		InstanceID:         "instance_test",
		SubscriptionID:     "sub_canceled",
		TargetEmail:        "test@example.com",
		JellyfinID:         "jf_user",
		Status:             paymentStatusSubscriptionCanceled,
		SubscriptionStatus: string(stripe.SubscriptionStatusCanceled),
	})

	reconciled := false
	skip := app.shouldSkipExpiredPaidUserWithReconcile("jf_user", UserExpiry{
		Expiry: time.Now().Add(-time.Hour),
	}, func() ReconcilePaymentsDTO {
		reconciled = true
		return ReconcilePaymentsDTO{}
	})

	if skip {
		t.Fatal("did not expect canceled Stripe subscription to skip expiry cleanup")
	}
	if reconciled {
		t.Fatal("did not expect terminal Stripe subscription to trigger reconciliation")
	}
}

func TestCancelingStripeSubscriptionSkipsCleanupUntilCancelDate(t *testing.T) {
	oldStripeEnabled := stripeEnabled
	stripeEnabled = true
	t.Cleanup(func() {
		stripeEnabled = oldStripeEnabled
	})

	app := newStripePaymentTestApp(t)
	app.storage.SetEmailsKey("jf_user", EmailAddress{Addr: "test@example.com"})
	app.storage.SetPaymentKey("cs_canceling", Payment{
		Provider:                      lm.Stripe,
		InstanceID:                    "instance_test",
		SubscriptionID:                "sub_canceling",
		TargetEmail:                   "test@example.com",
		JellyfinID:                    "jf_user",
		Status:                        paymentStatusSubscriptionCanceling,
		SubscriptionStatus:            string(stripe.SubscriptionStatusActive),
		SubscriptionCancelAt:          time.Now().Add(24 * time.Hour).Unix(),
		SubscriptionCancelAtPeriodEnd: true,
	})

	skip := app.shouldSkipExpiredPaidUserWithReconcile("jf_user", UserExpiry{
		Expiry: time.Now().Add(-time.Hour),
	}, func() ReconcilePaymentsDTO {
		return ReconcilePaymentsDTO{}
	})

	if !skip {
		t.Fatal("expected canceling Stripe subscription to skip cleanup before cancel date")
	}
}

func TestExpiredPaidInviteIsPreservedForRecovery(t *testing.T) {
	app := newStripePaymentTestApp(t)
	expired := time.Now().Add(-time.Hour)
	invite := Invite{
		Code:          "paid_invite",
		ValidTill:     expired,
		PaymentID:     "cs_paid",
		PaymentStatus: paymentStatusPaid,
		UserExpiry:    true,
		UserMonths:    1,
	}
	app.storage.SetInvitesKey(invite.Code, invite)
	app.storage.SetPaymentKey("cs_paid", Payment{
		Provider:    lm.Stripe,
		InviteCode:  invite.Code,
		TargetEmail: "test@example.com",
		Plan:        paymentPlanMonthly,
		Status:      paymentStatusEmailSent,
		PaidAt:      time.Now().Add(-2 * time.Hour),
	})

	app.deleteExpiredInvite(invite)

	preserved, ok := app.storage.GetInvitesKey(invite.Code)
	if !ok {
		t.Fatal("expected paid invite to be preserved")
	}
	if !preserved.ValidTill.Equal(expired) {
		t.Fatal("expected housekeeping to preserve, not extend, the paid invite")
	}
	payment, ok := app.storage.GetPaymentKey("cs_paid")
	if !ok {
		t.Fatal("expected payment to remain stored")
	}
	if payment.Status != paymentStatusNeedsReview {
		t.Fatalf("expected payment status %q, got %q", paymentStatusNeedsReview, payment.Status)
	}
	if payment.Error != paymentInviteExpiredNeedsReview {
		t.Fatalf("expected payment review reason %q, got %q", paymentInviteExpiredNeedsReview, payment.Error)
	}
}

func TestExpiredUnpaidInviteIsDeleted(t *testing.T) {
	app := newStripePaymentTestApp(t)
	invite := Invite{
		Code:      "unpaid_invite",
		ValidTill: time.Now().Add(-time.Hour),
	}
	app.storage.SetInvitesKey(invite.Code, invite)

	app.deleteExpiredInvite(invite)

	if _, ok := app.storage.GetInvitesKey(invite.Code); ok {
		t.Fatal("expected unpaid expired invite to be deleted")
	}
}

func TestExpiredPaidInviteRefreshesBeforeResend(t *testing.T) {
	app := newStripePaymentTestApp(t)
	invite := Invite{
		Code:       "paid_invite",
		ValidTill:  time.Now().Add(-time.Hour),
		UserExpiry: true,
	}
	payment := Payment{
		ID:           "cs_paid",
		Provider:     lm.Stripe,
		InviteCode:   invite.Code,
		Plan:         paymentPlanMonthly,
		AccessMonths: 1,
		Status:       paymentStatusNeedsReview,
		PaidAt:       time.Now().Add(-2 * time.Hour),
	}
	app.storage.SetInvitesKey(invite.Code, invite)

	refreshed := app.refreshPurchasedInviteForResend(payment, invite)

	if !refreshed.ValidTill.After(time.Now()) {
		t.Fatalf("expected refreshed invite expiry to be in the future, got %s", refreshed.ValidTill)
	}
	stored, ok := app.storage.GetInvitesKey(invite.Code)
	if !ok {
		t.Fatal("expected refreshed invite to be stored")
	}
	if !stored.ValidTill.Equal(refreshed.ValidTill) {
		t.Fatalf("expected stored invite expiry %s, got %s", refreshed.ValidTill, stored.ValidTill)
	}
}
