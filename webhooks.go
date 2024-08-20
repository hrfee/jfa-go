package main

import (
	"net/http"
	"net/url"
	"time"

	"github.com/hrfee/jfa-go/common"
	"github.com/hrfee/jfa-go/logger"
	lm "github.com/hrfee/jfa-go/logmessages"
)

type WebhookSender struct {
	httpClient     *http.Client
	timeoutHandler common.TimeoutHandler
	log            *logger.Logger
}

// SetTransport sets the http.Transport to use for requests. Can be used to set a proxy.
func (ws *WebhookSender) SetTransport(t *http.Transport) {
	ws.httpClient.Transport = t
}

func NewWebhookSender(timeoutHandler common.TimeoutHandler, log *logger.Logger) *WebhookSender {
	return &WebhookSender{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeoutHandler: timeoutHandler,
		log:            log,
	}
}

func (ws *WebhookSender) Send(uri string, payload any) (int, error) {
	_, status, err := common.Req(ws.httpClient, ws.timeoutHandler, http.MethodPost, uri, payload, url.Values{}, nil, true)
	ws.log.Printf(lm.WebhookRequest, uri, status, err)
	return status, err
}
