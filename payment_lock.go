package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"time"
)

const (
	paymentLockCookieName = "jfa_payment_lock"
	paymentLockMaxAge     = 24 * 3600
)

func newPaymentLockToken() (token, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, hashPaymentLockToken(token), nil
}

func hashPaymentLockToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (app *appContext) paymentForInviteUnlock(invite Invite) (Payment, bool) {
	if invite.PaymentID != "" {
		if payment, ok := app.storage.GetPaymentKey(invite.PaymentID); ok {
			return payment, true
		}
	}

	var best Payment
	found := false
	for _, payment := range app.storage.GetPayments() {
		if payment.InviteCode != invite.Code || payment.InviteLockHash == "" {
			continue
		}
		if !found || payment.Created.After(best.Created) {
			best = payment
			found = true
		}
	}
	return best, found
}

func (app *appContext) validPaidInvitePaymentLock(invite Invite, token string) bool {
	if !invite.RequiredPayment || invite.PaymentStatus != paymentStatusPaid {
		return true
	}
	if token == "" {
		return false
	}

	if payment, ok := app.paymentForInviteUnlock(invite); ok && payment.InviteLockHash != "" {
		return subtle.ConstantTimeCompare([]byte(payment.InviteLockHash), []byte(hashPaymentLockToken(token))) == 1
	}

	// Compatibility for paid unlocks created before random payment locks existed.
	return subtle.ConstantTimeCompare([]byte(token), []byte(invite.Code)) == 1
}

func (app *appContext) paidInvitePaymentLockFromCookie(gc interface {
	Cookie(string) (string, error)
}, invite Invite) bool {
	if !invite.RequiredPayment || invite.PaymentStatus != paymentStatusPaid {
		return true
	}
	token, err := gc.Cookie(paymentLockCookieName)
	if err != nil {
		return false
	}
	return app.validPaidInvitePaymentLock(invite, token)
}

func setPaymentLockCreated(payment *Payment) {
	if payment.InviteLockCreatedAt.IsZero() {
		payment.InviteLockCreatedAt = time.Now()
	}
}
