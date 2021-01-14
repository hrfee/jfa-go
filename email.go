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

// Mailgun client implements emailClient.
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

// SMTP supports SSL/TLS and STARTTLS; implements emailClient.
type SMTP struct {
	sslTLS bool
	server string
	port   int
	auth   smtp.Auth
}

func (sm *SMTP) send(address, fromName, fromAddr string, email *Email) error {
	e := jEmail.NewEmail()
	e.Subject = email.subject
	e.From = fmt.Sprintf("%s <%s>", fromName, fromAddr)
	e.To = []string{address}
	e.Text = []byte(email.text)
	e.HTML = []byte(email.html)
	server := fmt.Sprintf("%s:%d", sm.server, sm.port)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         sm.server,
	}
	var err error
	fmt.Println(server)
	// err = e.Send(server, sm.auth)
	if sm.sslTLS {
		err = e.SendWithTLS(server, sm.auth, tlsConfig)
	} else {
		err = e.SendWithStartTLS(server, sm.auth, tlsConfig)
	}
	return err
}

// Emailer contains the email sender, email content, and methods to construct message content.
type Emailer struct {
	fromAddr, fromName string
	lang               *EmailLang
	cLang              string
	sender             emailClient
}

// Email stores content.
type Email struct {
	subject    string
	html, text string
}

func (emailer *Emailer) formatExpiry(expiry time.Time, tzaware bool, datePattern, timePattern string) (d, t, expiresIn string) {
	d, _ = strtime.Strftime(expiry, datePattern)
	t, _ = strtime.Strftime(expiry, timePattern)
	currentTime := time.Now()
	if tzaware {
		currentTime = currentTime.UTC()
	}
	_, _, days, hours, minutes, _ := timeDiff(expiry, currentTime)
	if days != 0 {
		expiresIn += fmt.Sprintf("%dd ", days)
	}
	if hours != 0 {
		expiresIn += fmt.Sprintf("%dh ", hours)
	}
	if minutes != 0 {
		expiresIn += fmt.Sprintf("%dm ", minutes)
	}
	expiresIn = strings.TrimSuffix(expiresIn, " ")
	return
}

// NewEmailer configures and returns a new emailer.
func NewEmailer(app *appContext) *Emailer {
	emailer := &Emailer{
		fromAddr: app.config.Section("email").Key("address").String(),
		fromName: app.config.Section("email").Key("from").String(),
		lang:     &(app.storage.lang.Email),
		cLang:    app.storage.lang.chosenEmailLang,
	}
	method := app.config.Section("email").Key("method").String()
	if method == "smtp" {
		sslTls := false
		if app.config.Section("smtp").Key("encryption").String() == "ssl_tls" {
			sslTls = true
		}
		username := ""
		if u := app.config.Section("smtp").Key("username").MustString(""); u != "" {
			username = u
		} else {
			username = emailer.fromAddr
		}
		emailer.NewSMTP(app.config.Section("smtp").Key("server").String(), app.config.Section("smtp").Key("port").MustInt(465), username, app.config.Section("smtp").Key("password").String(), sslTls)
	} else if method == "mailgun" {
		emailer.NewMailgun(app.config.Section("mailgun").Key("api_url").String(), app.config.Section("mailgun").Key("api_key").String())
	}
	return emailer
}

// NewMailgun returns a Mailgun emailClient.
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

// NewSMTP returns an SMTP emailClient.
func (emailer *Emailer) NewSMTP(server string, port int, username, password string, sslTLS bool) {
	emailer.sender = &SMTP{
		auth:   smtp.PlainAuth("", username, password, server),
		server: server,
		port:   port,
		sslTLS: sslTLS,
	}
}

func (emailer *Emailer) constructInvite(code string, invite Invite, app *appContext) (*Email, error) {
	lang := emailer.cLang
	email := &Email{
		subject: app.config.Section("invite_emails").Key("subject").MustString(emailer.lang.get(lang, "inviteEmail", "title")),
	}
	expiry := invite.ValidTill
	d, t, expiresIn := emailer.formatExpiry(expiry, false, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	inviteLink := app.config.Section("invite_emails").Key("url_base").String()
	inviteLink = fmt.Sprintf("%s/%s", inviteLink, code)

	for _, key := range []string{"html", "text"} {
		fpath := app.config.Section("invite_emails").Key("email_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			return nil, err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"hello":              emailer.lang.get(lang, "inviteEmail", "hello"),
			"youHaveBeenInvited": emailer.lang.get(lang, "inviteEmail", "youHaveBeenInvited"),
			"toJoin":             emailer.lang.get(lang, "inviteEmail", "toJoin"),
			"inviteExpiry":       emailer.lang.format(lang, "inviteEmail", "inviteExpiry", d, t, expiresIn),
			"linkButton":         emailer.lang.get(lang, "inviteEmail", "linkButton"),
			"invite_link":        inviteLink,
			"message":            message,
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
	lang := emailer.cLang
	email := &Email{
		subject: emailer.lang.get(lang, "inviteExpiry", "title"),
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
			"inviteExpired":      emailer.lang.get(lang, "inviteExpiry", "inviteExpired"),
			"expiredAt":          emailer.lang.format(lang, "inviteExpiry", "expiredAt", "\""+code+"\"", expiry),
			"notificationNotice": emailer.lang.get(lang, "inviteExpiry", "notificationNotice"),
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
	lang := emailer.cLang
	email := &Email{
		subject: emailer.lang.get(lang, "userCreated", "title"),
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
			"aUserWasCreated":    emailer.lang.format(lang, "userCreated", "aUserWasCreated", "\""+code+"\""),
			"name":               emailer.lang.get(lang, "userCreated", "name"),
			"address":            emailer.lang.get(lang, "userCreated", "emailAddress"),
			"time":               emailer.lang.get(lang, "userCreated", "time"),
			"nameVal":            username,
			"addressVal":         tplAddress,
			"timeVal":            created,
			"notificationNotice": emailer.lang.get(lang, "userCreated", "notificationNotice"),
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

func (emailer *Emailer) constructReset(pwr PasswordReset, app *appContext) (*Email, error) {
	lang := emailer.cLang
	email := &Email{
		subject: emailer.lang.get(lang, "passwordReset", "title"),
	}
	d, t, expiresIn := emailer.formatExpiry(pwr.Expiry, true, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	for _, key := range []string{"html", "text"} {
		fpath := app.config.Section("password_resets").Key("email_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			return nil, err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"helloUser":                emailer.lang.format(lang, "passwordReset", "helloUser", pwr.Username),
			"someoneHasRequestedReset": emailer.lang.get(lang, "passwordReset", "someoneHasRequestedReset"),
			"ifItWasYou":               emailer.lang.get(lang, "passwordReset", "ifItWasYou"),
			"codeExpiry":               emailer.lang.format(lang, "passwordReset", "codeExpiry", d, t, expiresIn),
			"ifItWasNotYou":            emailer.lang.get(lang, "passwordReset", "ifItWasNotYou"),
			"pin":                      emailer.lang.get(lang, "passwordReset", "pin"),
			"pinVal":                   pwr.Pin,
			"message":                  message,
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

func (emailer *Emailer) constructDeleted(reason string, app *appContext) (*Email, error) {
	lang := emailer.cLang
	email := &Email{
		subject: emailer.lang.get(lang, "userDeleted", "title"),
	}
	for _, key := range []string{"html", "text"} {
		fpath := app.config.Section("deletion").Key("email_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			return nil, err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"yourAccountWasDeleted": emailer.lang.get(lang, "userDeleted", "yourAccountWasDeleted"),
			"reason":                emailer.lang.get(lang, "userDeleted", "reason"),
			"reasonVal":             reason,
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

// calls the send method in the underlying emailClient.
func (emailer *Emailer) send(address string, email *Email) error {
	return emailer.sender.send(address, emailer.fromName, emailer.fromAddr, email)
}
