package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	jEmail "github.com/jordan-wright/email"
	"github.com/knz/strtime"
	"github.com/mailgun/mailgun-go/v4"
)

// implements email sending, right now via smtp or mailgun.
type emailClient interface {
	send(address, fromName, fromAddr string, email *Email) error
}

type Mailgun struct {
	client *mailgun.MailgunImpl
}

func (mg *Mailgun) send(address, fromName, fromAddr string, email *Email) error {
	message := mg.client.NewMessage(
		fmt.Sprintf("%s <%s>", fromName, fromAddr),
		email.subject,
		email.text,
		address,
	)
	message.SetHtml(email.html)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	_, _, err := mg.client.Send(ctx, message)
	return err
}

type Smtp struct {
	sslTls       bool
	host, server string
	port         int
	auth         smtp.Auth
}

func (sm *Smtp) send(address, fromName, fromAddr string, email *Email) error {
	e := jEmail.NewEmail()
	e.Subject = email.subject
	e.From = fmt.Sprintf("%s <%s>", fromName, fromAddr)
	e.To = []string{address}
	e.Text = []byte(email.text)
	e.HTML = []byte(email.html)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         sm.host,
	}
	server := fmt.Sprintf("%s:%d", sm.server, sm.port)
	var err error
	if sm.sslTls {
		err = e.SendWithTLS(server, sm.auth, tlsConfig)
	} else {
		err = e.SendWithStartTLS(server, sm.auth, tlsConfig)
	}
	return err
}

// Emailer contains the email sender, email content, and methods to construct message content.
type Emailer struct {
	fromAddr, fromName string
	sender             emailClient
}

// Email stores content.
type Email struct {
	subject    string
	html, text string
}

func (email *Emailer) formatExpiry(expiry time.Time, tzaware bool, datePattern, timePattern string) (d, t, expires_in string) {
	d, _ = strtime.Strftime(expiry, datePattern)
	t, _ = strtime.Strftime(expiry, timePattern)
	current_time := time.Now()
	if tzaware {
		current_time = current_time.UTC()
	}
	_, _, days, hours, minutes, _ := timeDiff(expiry, current_time)
	if days != 0 {
		expires_in += fmt.Sprintf("%dd ", days)
	}
	if hours != 0 {
		expires_in += fmt.Sprintf("%dh ", hours)
	}
	if minutes != 0 {
		expires_in += fmt.Sprintf("%dm ", minutes)
	}
	expires_in = strings.TrimSuffix(expires_in, " ")
	return
}

func NewEmailer(app *appContext) *Emailer {
	emailer := &Emailer{
		fromAddr: app.config.Section("email").Key("address").String(),
		fromName: app.config.Section("email").Key("from").String(),
	}
	method := app.config.Section("email").Key("method").String()
	if method == "smtp" {
		sslTls := false
		if app.config.Section("smtp").Key("encryption").String() == "ssl_tls" {
			sslTls = true
		}
		emailer.NewSMTP(app.config.Section("smtp").Key("server").String(), app.config.Section("smtp").Key("port").MustInt(465), app.config.Section("smtp").Key("password").String(), app.host, sslTls)
	} else if method == "mailgun" {
		emailer.NewMailgun(app.config.Section("mailgun").Key("api_url").String(), app.config.Section("mailgun").Key("api_key").String())
	}
	return emailer
}

func (emailer *Emailer) NewMailgun(url, key string) {
	sender := &Mailgun{
		client: mailgun.NewMailgun(strings.Split(emailer.fromAddr, "@")[1], key),
	}
	// Mailgun client takes the base url, so we need to trim off the end (e.g 'v3/messages'
	if strings.Contains(url, "messages") {
		url = url[0:strings.LastIndex(url, "/")]
		url = url[0:strings.LastIndex(url, "/")]
	}
	sender.client.SetAPIBase(url)
	emailer.sender = sender
}

func (emailer *Emailer) NewSMTP(server string, port int, password, host string, sslTls bool) {
	emailer.sender = &Smtp{
		auth:   smtp.PlainAuth("", emailer.fromAddr, password, host),
		server: server,
		host:   host,
		port:   port,
		sslTls: sslTls,
	}
}

func (emailer *Emailer) constructInvite(code string, invite Invite, app *appContext) (*Email, error) {
	email := &Email{
		subject: app.config.Section("invite_emails").Key("subject").String(),
	}
	expiry := invite.ValidTill
	d, t, expires_in := emailer.formatExpiry(expiry, false, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	invite_link := app.config.Section("invite_emails").Key("url_base").String()
	invite_link = fmt.Sprintf("%s/%s", invite_link, code)

	for _, key := range []string{"html", "text"} {
		fpath := app.config.Section("invite_emails").Key("email_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			return nil, err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"expiry_date": d,
			"expiry_time": t,
			"expires_in":  expires_in,
			"invite_link": invite_link,
			"message":     message,
		})
		if err != nil {
			return nil, err
		}
		if key == "html" {
			email.html = tplData.String()
		} else {
			email.text = tplData.String()
		}
	}
	return email, nil
}

func (emailer *Emailer) constructExpiry(code string, invite Invite, app *appContext) (*Email, error) {
	email := &Email{
		subject: "Notice: Invite expired",
	}
	expiry := app.formatDatetime(invite.ValidTill)
	for _, key := range []string{"html", "text"} {
		fpath := app.config.Section("notifications").Key("expiry_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			return nil, err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"code":   code,
			"expiry": expiry,
		})
		if err != nil {
			return nil, err
		}
		if key == "html" {
			email.html = tplData.String()
		} else {
			email.text = tplData.String()
		}
	}
	return email, nil
}

func (emailer *Emailer) constructCreated(code, username, address string, invite Invite, app *appContext) (*Email, error) {
	email := &Email{
		subject: "Notice: User created",
	}
	created := app.formatDatetime(invite.Created)
	var tplAddress string
	if app.config.Section("email").Key("no_username").MustBool(false) {
		tplAddress = "n/a"
	} else {
		tplAddress = address
	}
	for _, key := range []string{"html", "text"} {
		fpath := app.config.Section("notifications").Key("created_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			return nil, err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"code":     code,
			"username": username,
			"address":  tplAddress,
			"time":     created,
		})
		if err != nil {
			return nil, err
		}
		if key == "html" {
			email.html = tplData.String()
		} else {
			email.text = tplData.String()
		}
	}
	return email, nil
}

func (emailer *Emailer) constructReset(pwr Pwr, app *appContext) (*Email, error) {
	email := &Email{
		subject: app.config.Section("password_resets").Key("subject").MustString("Password reset - Jellyfin"),
	}
	d, t, expires_in := emailer.formatExpiry(pwr.Expiry, true, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	for _, key := range []string{"html", "text"} {
		fpath := app.config.Section("password_resets").Key("email_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			return nil, err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"username":    pwr.Username,
			"expiry_date": d,
			"expiry_time": t,
			"expires_in":  expires_in,
			"pin":         pwr.Pin,
			"message":     message,
		})
		if err != nil {
			return nil, err
		}
		if key == "html" {
			email.html = tplData.String()
		} else {
			email.text = tplData.String()
		}
	}
	return email, nil
}

func (emailer *Emailer) send(address string, email *Email) error {
	return emailer.sender.send(address, emailer.fromName, emailer.fromAddr, email)
}
