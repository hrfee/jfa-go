package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	jEmail "github.com/jordan-wright/email"
	"github.com/knz/strtime"
	"github.com/mailgun/mailgun-go/v4"
)

// implements email sending, right now via smtp or mailgun.
type emailClient interface {
	send(fromName, fromAddr string, email *Email, address ...string) error
}

// Mailgun client implements emailClient.
type Mailgun struct {
	client *mailgun.MailgunImpl
}

func (mg *Mailgun) send(fromName, fromAddr string, email *Email, address ...string) error {
	message := mg.client.NewMessage(
		fmt.Sprintf("%s <%s>", fromName, fromAddr),
		email.subject,
		email.text,
	)
	for _, a := range address {
		// Adding variable tells mailgun to do a batch send, so users don't see other recipients.
		message.AddRecipientAndVariables(a, map[string]interface{}{"unique_id": a})
	}
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

func (sm *SMTP) send(fromName, fromAddr string, email *Email, address ...string) error {
	server := fmt.Sprintf("%s:%d", sm.server, sm.port)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         sm.server,
	}
	from := fmt.Sprintf("%s <%s>", fromName, fromAddr)
	var wg sync.WaitGroup
	var err error
	for _, addr := range address {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			e := jEmail.NewEmail()
			e.Subject = email.subject
			e.From = from
			e.Text = []byte(email.text)
			e.HTML = []byte(email.html)
			e.To = []string{addr}
			if sm.sslTLS {
				err = e.SendWithTLS(server, sm.auth, tlsConfig)
			} else {
				err = e.SendWithStartTLS(server, sm.auth, tlsConfig)
			}
		}(addr)
	}
	wg.Wait()
	return err
}

// Emailer contains the email sender, email content, and methods to construct message content.
type Emailer struct {
	fromAddr, fromName string
	lang               emailLang
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
		lang:     app.storage.lang.Email[app.storage.lang.chosenEmailLang],
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
	// Mailgun client takes the base url, so we need to trim off the end (e.g 'v3/messages')
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

func (emailer *Emailer) construct(app *appContext, section, keyFragment string, data map[string]interface{}) (html, text string, err error) {
	var tpl *template.Template
	for _, key := range []string{"html", "text"} {
		filesystem, fpath := app.GetPath(section, keyFragment+key)
		tpl, err = template.ParseFS(filesystem, fpath)
		if err != nil {
			return
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, data)
		if err != nil {
			return
		}
		if key == "html" {
			html = tplData.String()
		} else {
			text = tplData.String()
		}
	}
	return
}

func (emailer *Emailer) constructConfirmation(code, username, key string, app *appContext) (*Email, error) {
	email := &Email{
		subject: app.config.Section("email_confirmation").Key("subject").MustString(emailer.lang.EmailConfirmation.get("title")),
	}
	message := app.config.Section("email").Key("message").String()
	inviteLink := app.config.Section("invite_emails").Key("url_base").String()
	inviteLink = fmt.Sprintf("%s/%s?key=%s", inviteLink, code, key)
	var err error
	email.html, email.text, err = emailer.construct(app, "email_confirmation", "email_", map[string]interface{}{
		"helloUser":     emailer.lang.Strings.format("helloUser", username),
		"clickBelow":    emailer.lang.EmailConfirmation.get("clickBelow"),
		"ifItWasNotYou": emailer.lang.Strings.get("ifItWasNotYou"),
		"urlVal":        inviteLink,
		"confirmEmail":  emailer.lang.EmailConfirmation.get("confirmEmail"),
		"message":       message,
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructAnnouncement(subject, md string, app *appContext) (*Email, error) {
	email := &Email{subject: subject}
	renderer := html.NewRenderer(html.RendererOptions{Flags: html.Smartypants})
	html := markdown.ToHTML([]byte(md), nil, renderer)
	message := app.config.Section("email").Key("message").String()
	var err error
	email.html, email.text, err = emailer.construct(app, "announcement_email", "email_", map[string]interface{}{
		"text":    template.HTML(html),
		"message": message,
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructInvite(code string, invite Invite, app *appContext) (*Email, error) {
	email := &Email{
		subject: app.config.Section("email_confirmation").Key("subject").MustString(emailer.lang.InviteEmail.get("title")),
	}
	expiry := invite.ValidTill
	d, t, expiresIn := emailer.formatExpiry(expiry, false, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	inviteLink := app.config.Section("invite_emails").Key("url_base").String()
	inviteLink = fmt.Sprintf("%s/%s", inviteLink, code)
	var err error
	email.html, email.text, err = emailer.construct(app, "invite_emails", "email_", map[string]interface{}{
		"hello":              emailer.lang.InviteEmail.get("hello"),
		"youHaveBeenInvited": emailer.lang.InviteEmail.get("youHaveBeenInvited"),
		"toJoin":             emailer.lang.InviteEmail.get("toJoin"),
		"inviteExpiry":       emailer.lang.InviteEmail.format("inviteExpiry", d, t, expiresIn),
		"linkButton":         emailer.lang.InviteEmail.get("linkButton"),
		"invite_link":        inviteLink,
		"message":            message,
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructExpiry(code string, invite Invite, app *appContext) (*Email, error) {
	email := &Email{
		subject: emailer.lang.InviteExpiry.get("title"),
	}
	expiry := app.formatDatetime(invite.ValidTill)
	var err error
	email.html, email.text, err = emailer.construct(app, "notifications", "expiry_", map[string]interface{}{
		"inviteExpired":      emailer.lang.InviteExpiry.get("inviteExpired"),
		"expiredAt":          emailer.lang.InviteExpiry.format("expiredAt", "\""+code+"\"", expiry),
		"notificationNotice": emailer.lang.InviteExpiry.get("notificationNotice"),
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructCreated(code, username, address string, invite Invite, app *appContext) (*Email, error) {
	email := &Email{
		subject: emailer.lang.UserCreated.get("title"),
	}
	created := app.formatDatetime(invite.Created)
	var tplAddress string
	if app.config.Section("email").Key("no_username").MustBool(false) {
		tplAddress = "n/a"
	} else {
		tplAddress = address
	}
	var err error
	email.html, email.text, err = emailer.construct(app, "notifications", "created_", map[string]interface{}{
		"aUserWasCreated":    emailer.lang.UserCreated.format("aUserWasCreated", "\""+code+"\""),
		"name":               emailer.lang.Strings.get("name"),
		"address":            emailer.lang.Strings.get("emailAddress"),
		"time":               emailer.lang.UserCreated.get("time"),
		"nameVal":            username,
		"addressVal":         tplAddress,
		"timeVal":            created,
		"notificationNotice": emailer.lang.UserCreated.get("notificationNotice"),
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructReset(pwr PasswordReset, app *appContext) (*Email, error) {
	email := &Email{
		subject: app.config.Section("password_resets").Key("subject").MustString(emailer.lang.PasswordReset.get("title")),
	}
	d, t, expiresIn := emailer.formatExpiry(pwr.Expiry, true, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	var err error
	email.html, email.text, err = emailer.construct(app, "password_resets", "email_", map[string]interface{}{
		"helloUser":                emailer.lang.Strings.format("helloUser", pwr.Username),
		"someoneHasRequestedReset": emailer.lang.PasswordReset.get("someoneHasRequestedReset"),
		"ifItWasYou":               emailer.lang.PasswordReset.get("ifItWasYou"),
		"codeExpiry":               emailer.lang.PasswordReset.format("codeExpiry", d, t, expiresIn),
		"ifItWasNotYou":            emailer.lang.Strings.get("ifItWasNotYou"),
		"pin":                      emailer.lang.PasswordReset.get("pin"),
		"pinVal":                   pwr.Pin,
		"message":                  message,
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructDeleted(reason string, app *appContext) (*Email, error) {
	email := &Email{
		subject: app.config.Section("deletion").Key("subject").MustString(emailer.lang.UserDeleted.get("title")),
	}
	var err error
	email.html, email.text, err = emailer.construct(app, "deletion", "email_", map[string]interface{}{
		"yourAccountWasDeleted": emailer.lang.UserDeleted.get("yourAccountWasDeleted"),
		"reason":                emailer.lang.UserDeleted.get("reason"),
		"reasonVal":             reason,
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructWelcome(username string, app *appContext) (*Email, error) {
	email := &Email{
		subject: app.config.Section("welcome_email").Key("subject").MustString(emailer.lang.WelcomeEmail.get("title")),
	}
	var err error
	email.html, email.text, err = emailer.construct(app, "welcome_email", "email_", map[string]interface{}{
		"welcome":         emailer.lang.WelcomeEmail.get("welcome"),
		"youCanLoginWith": emailer.lang.WelcomeEmail.get("youCanLoginWith"),
		"jellyfinURL":     emailer.lang.WelcomeEmail.get("jellyfinURL"),
		"jellyfinURLVal":  app.config.Section("jellyfin").Key("public_server").String(),
		"username":        emailer.lang.Strings.get("username"),
		"usernameVal":     username,
		"message":         app.config.Section("email").Key("message").String(),
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

// calls the send method in the underlying emailClient.
func (emailer *Emailer) send(email *Email, address ...string) error {
	return emailer.sender.send(emailer.fromName, emailer.fromAddr, email, address...)
}
