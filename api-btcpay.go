package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hrfee/jfa-go/btcpay"
	lm "github.com/hrfee/jfa-go/logmessages"
)

var btcpayClient *btcpay.Client

// InitBTCPay initializes the BTCPay Server client from config.
func InitBTCPay(config *Config) {
	server := config.Section("btcpay").Key("server").String()
	apiKey := config.Section("btcpay").Key("api_key").String()
	storeID := config.Section("btcpay").Key("store_id").String()
	webhookSecret := config.Section("btcpay").Key("webhook_secret").String()

	if server == "" || apiKey == "" || storeID == "" {
		btcpayEnabled = false
		return
	}

	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "https://" + server
	}

	btcpayClient = btcpay.NewClient(server, apiKey, storeID, webhookSecret)
	btcpayEnabled = true
}

type btcpayCheckoutDTO struct {
	Email string `json:"email" binding:"required,email"`
	Plan  string `json:"plan" binding:"required"`
}

// @Summary Create a BTCPay checkout invoice (Pay-to-Generate).
// @Produce json
// @Param body body btcpayCheckoutDTO true "Checkout Request"
// @Success 200 {object} stringResponse
// @Router /btcpay/create-checkout [post]
func (app *appContext) PostBTCPayCreateCheckout(gc *gin.Context) {
	if !btcpayEnabled {
		respond(400, "BTCPay disabled", gc)
		return
	}

	var req btcpayCheckoutDTO
	if err := gc.ShouldBindJSON(&req); err != nil {
		respond(400, "Invalid request: "+err.Error(), gc)
		return
	}

	if req.Plan == "Monthly" {
		if userID, _, found := app.findUserByEmail(req.Email); found {
			user, err := app.jf.UserByID(userID, false)
			if err == nil && !user.Policy.IsDisabled {
				expiry := time.Now()
				if userExpiry, ok := app.storage.GetUserExpiryKey(userID); ok {
					expiry = userExpiry.Expiry
				}
				if expiry.After(time.Now()) {
					app.info.Printf(lm.BTCPayBlockedDuplicate, userID, req.Email)
					respond(409, "You already have an active subscription.", gc)
					return
				}
			}
		}
	}

	currency := app.config.Section("btcpay").Key("price_currency").MustString("USD")
	priceMonthly := app.config.Section("btcpay").Key("price_monthly").MustFloat64(2.00)

	var priceAmount float64
	var profileName = "Default"

	if req.Plan == "Monthly" {
		priceAmount = priceMonthly
	} else {
		req.Plan = "Standard"
		priceAmount = priceMonthly // fallback; extend as needed
	}

	refID := "btcpay-" + strconv.FormatInt(time.Now().Unix(), 10)

	baseURL := ExternalURI(gc)
	successURL := fmt.Sprintf("%s/payment/success", baseURL)

	invoice, err := btcpayClient.CreateInvoice(btcpay.InvoiceRequest{
		Amount:   priceAmount,
		Currency: currency,
		Metadata: map[string]string{
			"target_email": req.Email,
			"plan":         req.Plan,
			"profile":      profileName,
			"ref_id":       refID,
		},
		Checkout: &btcpay.InvoiceCheckout{
			RedirectURL:           successURL,
			RedirectAutomatically: true,
		},
	})
	if err != nil {
		app.err.Printf(lm.FailedCreateBTCPayInvoice, err)
		respond(500, "Failed to create BTCPay invoice", gc)
		return
	}

	app.info.Printf(lm.BTCPayInvoiceCreated, invoice.ID, req.Email)
	gc.JSON(200, stringResponse{Response: invoice.CheckoutLink})
}

// @Summary Handle BTCPay Server Webhooks
// @Router /btcpay/webhook [post]
func (app *appContext) BTCPayWebhook(gc *gin.Context) {
	if !btcpayEnabled {
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

	verifySignature := app.config.Section("btcpay").Key("verify_signature").MustBool(true)
	if verifySignature {
		sigHeader := gc.GetHeader("BTCPay-Sig")
		if !btcpayClient.VerifyWebhookSignature(payload, sigHeader) {
			app.err.Printf(lm.BTCPayWebhookError, fmt.Errorf("invalid webhook signature"))
			gc.AbortWithStatus(403)
			return
		}
	}

	event, err := btcpay.ParseWebhookEvent(payload)
	if err != nil {
		app.err.Printf(lm.BTCPayWebhookError, err)
		gc.AbortWithStatus(400)
		return
	}

	app.info.Printf(lm.BTCPayWebhookReceived, event.Type, event.InvoiceID)

	switch event.Type {
	case "InvoiceSettled":
		app.handleBTCPayInvoiceSettled(event)
	case "InvoiceExpired":
		app.debug.Printf(lm.BTCPayInvoiceExpired, event.InvoiceID)
	case "InvoiceInvalid":
		app.err.Printf(lm.BTCPayInvoiceInvalid, event.InvoiceID)
	}

	gc.Status(200)
}

func (app *appContext) handleBTCPayInvoiceSettled(event *btcpay.WebhookEvent) {
	invoice, err := btcpayClient.GetInvoice(event.InvoiceID)
	if err != nil {
		app.err.Printf(lm.BTCPayWebhookError, fmt.Errorf("failed to get invoice %s: %w", event.InvoiceID, err))
		return
	}

	targetEmail := invoice.Metadata["target_email"]
	plan := invoice.Metadata["plan"]
	profile := invoice.Metadata["profile"]

	if targetEmail == "" {
		app.err.Printf(lm.BTCPayWebhookError, fmt.Errorf("invoice %s has no target_email in metadata", event.InvoiceID))
		return
	}

	app.info.Printf(lm.BTCPayPaymentReceived, plan, targetEmail, event.InvoiceID)

	existingUserID, existingEmail, found := app.findUserByEmail(targetEmail)
	if found {
		app.info.Printf(lm.ExistingUserFound, targetEmail, existingUserID)

		existingEmail.Label = "BTCPay Invoice: " + event.InvoiceID
		app.storage.SetEmailsKey(existingUserID, existingEmail)

		expiry := time.Now()
		lastTx := ""
		if userExpiry, ok := app.storage.GetUserExpiryKey(existingUserID); ok {
			expiry = userExpiry.Expiry
			lastTx = userExpiry.LastTransactionID
		}

		if lastTx == event.InvoiceID {
			app.info.Printf(lm.BTCPayInvoiceAlreadyProcessed, event.InvoiceID, existingUserID)
			return
		}

		if expiry.Before(time.Now()) {
			expiry = time.Now()
		}
		var newExpiry time.Time
		if plan == "Monthly" {
			newExpiry = expiry.AddDate(0, 1, 0)
		} else {
			newExpiry = expiry.AddDate(10, 0, 0)
		}
		app.storage.SetUserExpiryKey(existingUserID, UserExpiry{Expiry: newExpiry, LastTransactionID: event.InvoiceID})

		if paramsUser, err := app.jf.UserByID(existingUserID, false); err == nil {
			if err, _, _ = app.SetUserDisabled(paramsUser, false); err != nil {
				app.err.Printf(lm.FailedReEnableUser, existingUserID, err)
			}
			app.InvalidateUserCaches()
		}

		app.info.Printf(lm.UserReactivated, existingUserID, newExpiry)
		return
	}

	inviteCode := GenerateInviteCode()
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
		Label:         "BTCPay " + plan + " by " + targetEmail,
		UserLabel:     "Purchased via BTCPay",
		RemainingUses: 1,
		Profile:       profile,
		SendTo:        targetEmail,
	}

	if plan == "Monthly" {
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
