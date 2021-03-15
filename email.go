package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"os"
	"strings"
	"sync"
	textTemplate "text/template"
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
		email.Subject,
		email.Text,
	)
	for _, a := range address {
		// Adding variable tells mailgun to do a batch send, so users don't see other recipients.
		message.AddRecipientAndVariables(a, map[string]interface{}{"unique_id": a})
	}
	message.SetHtml(email.HTML)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	_, _, err := mg.client.Send(ctx, message)
	return err
}

// SMTP supports SSL/TLS and STARTTLS; implements emailClient.
type SMTP struct {
	sslTLS    bool
	server    string
	port      int
	auth      smtp.Auth
	tlsConfig *tls.Config
}

func (sm *SMTP) send(fromName, fromAddr string, email *Email, address ...string) error {
	server := fmt.Sprintf("%s:%d", sm.server, sm.port)
	from := fmt.Sprintf("%s <%s>", fromName, fromAddr)
	var wg sync.WaitGroup
	var err error
	for _, addr := range address {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			e := jEmail.NewEmail()
			e.Subject = email.Subject
			e.From = from
			e.Text = []byte(email.Text)
			e.HTML = []byte(email.HTML)
			e.To = []string{addr}
			if sm.sslTLS {
				err = e.SendWithTLS(server, sm.auth, sm.tlsConfig)
			} else {
				err = e.SendWithStartTLS(server, sm.auth, sm.tlsConfig)
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
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
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
		err := emailer.NewSMTP(app.config.Section("smtp").Key("server").String(), app.config.Section("smtp").Key("port").MustInt(465), username, app.config.Section("smtp").Key("password").String(), sslTls, app.config.Section("smtp").Key("ssl_cert").MustString(""))
		if err != nil {
			app.err.Printf("Error while initiating SMTP mailer: %v", err)
		}
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
func (emailer *Emailer) NewSMTP(server string, port int, username, password string, sslTLS bool, certPath string) (err error) {
	rootCAs, err := x509.SystemCertPool()
	if rootCAs == nil || err != nil {
		rootCAs = x509.NewCertPool()
	}
	if certPath != "" {
		var cert []byte
		cert, err = os.ReadFile(certPath)
		if rootCAs.AppendCertsFromPEM(cert) == false {
			err = errors.New("Failed to append cert to pool")
		}
	}
	emailer.sender = &SMTP{
		auth:   smtp.PlainAuth("", username, password, server),
		server: server,
		port:   port,
		sslTLS: sslTLS,
		tlsConfig: &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         server,
			RootCAs:            rootCAs,
		},
	}
	return
}

type templ interface {
	Execute(wr io.Writer, data interface{}) error
}

func (emailer *Emailer) construct(app *appContext, section, keyFragment string, data map[string]interface{}) (html, text string, err error) {
	var tpl templ
	if substituteStrings == "" {
		data["jellyfin"] = "Jellyfin"
	} else {
		data["jellyfin"] = substituteStrings
	}
	var keys []string
	if app.config.Section("email").Key("plaintext").MustBool(false) {
		keys = []string{"text"}
		text = ""
	} else {
		keys = []string{"html", "text"}
	}
	for _, key := range keys {
		filesystem, fpath := app.GetPath(section, keyFragment+key)
		if key == "html" {
			tpl, err = template.ParseFS(filesystem, fpath)
		} else {
			tpl, err = textTemplate.ParseFS(filesystem, fpath)
		}
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

func (emailer *Emailer) confirmationValues(code, username, key string, app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"clickBelow":    emailer.lang.EmailConfirmation.get("clickBelow"),
		"ifItWasNotYou": emailer.lang.Strings.get("ifItWasNotYou"),
		"confirmEmail":  emailer.lang.EmailConfirmation.get("confirmEmail"),
		"message":       "",
		"username":      username,
	}
	if noSub {
		template["helloUser"] = emailer.lang.Strings.get("helloUser")
		empty := []string{"confirmationURL"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		message := app.config.Section("email").Key("message").String()
		inviteLink := app.config.Section("invite_emails").Key("url_base").String()
		inviteLink = fmt.Sprintf("%s/%s?key=%s", inviteLink, code, key)
		template["helloUser"] = emailer.lang.Strings.template("helloUser", tmpl{"username": username})
		template["confirmationURL"] = inviteLink
		template["message"] = message
	}
	return template
}

func (emailer *Emailer) constructConfirmation(code, username, key string, app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: app.config.Section("email_confirmation").Key("subject").MustString(emailer.lang.EmailConfirmation.get("title")),
	}
	var err error
	template := emailer.confirmationValues(code, username, key, app, noSub)
	if app.storage.customEmails.EmailConfirmation.Enabled {
		content := app.storage.customEmails.EmailConfirmation.Content
		for _, v := range app.storage.customEmails.EmailConfirmation.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "email_confirmation", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) constructTemplate(subject, md string, app *appContext) (*Email, error) {
	email := &Email{Subject: subject}
	renderer := html.NewRenderer(html.RendererOptions{Flags: html.Smartypants})
	html := markdown.ToHTML([]byte(md), nil, renderer)
	text := stripMarkdown(md)
	message := app.config.Section("email").Key("message").String()
	var err error
	email.HTML, email.Text, err = emailer.construct(app, "template_email", "email_", map[string]interface{}{
		"text":      template.HTML(html),
		"plaintext": text,
		"message":   message,
	})
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) inviteValues(code string, invite Invite, app *appContext, noSub bool) map[string]interface{} {
	expiry := invite.ValidTill
	d, t, expiresIn := emailer.formatExpiry(expiry, false, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	inviteLink := app.config.Section("invite_emails").Key("url_base").String()
	inviteLink = fmt.Sprintf("%s/%s", inviteLink, code)
	template := map[string]interface{}{
		"hello":              emailer.lang.InviteEmail.get("hello"),
		"youHaveBeenInvited": emailer.lang.InviteEmail.get("youHaveBeenInvited"),
		"toJoin":             emailer.lang.InviteEmail.get("toJoin"),
		"linkButton":         emailer.lang.InviteEmail.get("linkButton"),
		"message":            "",
		"date":               d,
		"time":               t,
		"expiresInMinutes":   expiresIn,
	}
	if noSub {
		template["inviteExpiry"] = emailer.lang.InviteEmail.get("inviteExpiry")
		empty := []string{"inviteURL"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["inviteExpiry"] = emailer.lang.InviteEmail.template("inviteExpiry", tmpl{"date": d, "time": t, "expiresInMinutes": expiresIn})
		template["inviteURL"] = inviteLink
		template["message"] = message
	}
	return template
}

func (emailer *Emailer) constructInvite(code string, invite Invite, app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: app.config.Section("email_confirmation").Key("subject").MustString(emailer.lang.InviteEmail.get("title")),
	}
	template := emailer.inviteValues(code, invite, app, noSub)
	var err error
	if app.storage.customEmails.InviteEmail.Enabled {
		content := app.storage.customEmails.InviteEmail.Content
		for _, v := range app.storage.customEmails.InviteEmail.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "invite_emails", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) expiryValues(code string, invite Invite, app *appContext, noSub bool) map[string]interface{} {
	expiry := app.formatDatetime(invite.ValidTill)
	template := map[string]interface{}{
		"inviteExpired":      emailer.lang.InviteExpiry.get("inviteExpired"),
		"notificationNotice": emailer.lang.InviteExpiry.get("notificationNotice"),
		"code":               "\"" + code + "\"",
		"time":               expiry,
	}
	if noSub {
		template["expiredAt"] = emailer.lang.InviteExpiry.get("expiredAt")
	} else {
		template["expiredAt"] = emailer.lang.InviteExpiry.template("expiredAt", tmpl{"code": template["code"].(string), "time": template["time"].(string)})
	}
	return template
}

func (emailer *Emailer) constructExpiry(code string, invite Invite, app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: emailer.lang.InviteExpiry.get("title"),
	}
	var err error
	template := emailer.expiryValues(code, invite, app, noSub)
	if app.storage.customEmails.InviteExpiry.Enabled {
		content := app.storage.customEmails.InviteExpiry.Content
		for _, v := range app.storage.customEmails.InviteExpiry.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "notifications", "expiry_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) createdValues(code, username, address string, invite Invite, app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"nameString":         emailer.lang.Strings.get("name"),
		"addressString":      emailer.lang.Strings.get("emailAddress"),
		"timeString":         emailer.lang.UserCreated.get("time"),
		"notificationNotice": "",
		"code":               "\"" + code + "\"",
	}
	if noSub {
		template["aUserWasCreated"] = emailer.lang.UserCreated.get("aUserWasCreated")
		empty := []string{"name", "address", "time"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		created := app.formatDatetime(invite.Created)
		var tplAddress string
		if app.config.Section("email").Key("no_username").MustBool(false) {
			tplAddress = "n/a"
		} else {
			tplAddress = address
		}
		template["aUserWasCreated"] = emailer.lang.UserCreated.template("aUserWasCreated", tmpl{"code": template["code"].(string)})
		template["name"] = username
		template["address"] = tplAddress
		template["time"] = created
		template["notificationNotice"] = emailer.lang.UserCreated.get("notificationNotice")
	}
	return template
}

func (emailer *Emailer) constructCreated(code, username, address string, invite Invite, app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: emailer.lang.UserCreated.get("title"),
	}
	template := emailer.createdValues(code, username, address, invite, app, noSub)
	var err error
	if app.storage.customEmails.UserCreated.Enabled {
		content := app.storage.customEmails.UserCreated.Content
		for _, v := range app.storage.customEmails.UserCreated.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "notifications", "created_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) resetValues(pwr PasswordReset, app *appContext, noSub bool) map[string]interface{} {
	d, t, expiresIn := emailer.formatExpiry(pwr.Expiry, true, app.datePattern, app.timePattern)
	message := app.config.Section("email").Key("message").String()
	template := map[string]interface{}{
		"someoneHasRequestedReset": emailer.lang.PasswordReset.get("someoneHasRequestedReset"),
		"ifItWasYou":               emailer.lang.PasswordReset.get("ifItWasYou"),
		"ifItWasNotYou":            emailer.lang.Strings.get("ifItWasNotYou"),
		"pinString":                emailer.lang.PasswordReset.get("pin"),
		"message":                  "",
		"username":                 pwr.Username,
		"date":                     d,
		"time":                     t,
		"expiresInMinutes":         expiresIn,
	}
	if noSub {
		template["helloUser"] = emailer.lang.Strings.get("helloUser")
		template["codeExpiry"] = emailer.lang.PasswordReset.get("codeExpiry")
		empty := []string{"pin"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["helloUser"] = emailer.lang.Strings.template("helloUser", tmpl{"username": pwr.Username})
		template["codeExpiry"] = emailer.lang.PasswordReset.template("codeExpiry", tmpl{"date": d, "time": t, "expiresInMinutes": expiresIn})
		template["pin"] = pwr.Pin
		template["message"] = message
	}
	return template
}

func (emailer *Emailer) constructReset(pwr PasswordReset, app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: app.config.Section("password_resets").Key("subject").MustString(emailer.lang.PasswordReset.get("title")),
	}
	template := emailer.resetValues(pwr, app, noSub)
	var err error
	if app.storage.customEmails.PasswordReset.Enabled {
		content := app.storage.customEmails.PasswordReset.Content
		for _, v := range app.storage.customEmails.PasswordReset.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "password_resets", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) deletedValues(reason string, app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"yourAccountWasDeleted": emailer.lang.UserDeleted.get("yourAccountWasDeleted"),
		"reasonString":          emailer.lang.UserDeleted.get("reason"),
		"message":               "",
	}
	if noSub {
		empty := []string{"reason"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["reason"] = reason
		template["message"] = app.config.Section("email").Key("message").String()
	}
	return template
}

func (emailer *Emailer) constructDeleted(reason string, app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: app.config.Section("deletion").Key("subject").MustString(emailer.lang.UserDeleted.get("title")),
	}
	var err error
	template := emailer.deletedValues(reason, app, noSub)
	if app.storage.customEmails.UserDeleted.Enabled {
		content := app.storage.customEmails.UserDeleted.Content
		for _, v := range app.storage.customEmails.UserDeleted.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "deletion", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) welcomeValues(username string, app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"welcome":           emailer.lang.WelcomeEmail.get("welcome"),
		"youCanLoginWith":   emailer.lang.WelcomeEmail.get("youCanLoginWith"),
		"jellyfinURLString": emailer.lang.WelcomeEmail.get("jellyfinURL"),
		"usernameString":    emailer.lang.Strings.get("username"),
		"message":           "",
	}
	if noSub {
		empty := []string{"jellyfinURL", "username"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["jellyfinURL"] = app.config.Section("jellyfin").Key("public_server").String()
		template["username"] = username
		template["message"] = app.config.Section("email").Key("message").String()
	}
	return template
}

func (emailer *Emailer) constructWelcome(username string, app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: app.config.Section("welcome_email").Key("subject").MustString(emailer.lang.WelcomeEmail.get("title")),
	}
	var err error
	template := emailer.welcomeValues(username, app, noSub)
	if app.storage.customEmails.WelcomeEmail.Enabled {
		content := app.storage.customEmails.WelcomeEmail.Content
		for _, v := range app.storage.customEmails.WelcomeEmail.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "welcome_email", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) userExpiredValues(app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"yourAccountHasExpired": emailer.lang.UserExpired.get("yourAccountHasExpired"),
		"contactTheAdmin":       emailer.lang.UserExpired.get("contactTheAdmin"),
		"message":               "",
	}
	if !noSub {
		template["message"] = app.config.Section("email").Key("message").String()
	}
	return template
}

func (emailer *Emailer) constructUserExpired(app *appContext, noSub bool) (*Email, error) {
	email := &Email{
		Subject: app.config.Section("user_expiry").Key("subject").MustString(emailer.lang.UserExpired.get("title")),
	}
	var err error
	template := emailer.userExpiredValues(app, noSub)
	if app.storage.customEmails.UserExpired.Enabled {
		content := app.storage.customEmails.UserExpired.Content
		for _, v := range app.storage.customEmails.UserExpired.Variables {
			replaceWith, ok := template[v[1:len(v)-1]]
			if ok {
				content = strings.ReplaceAll(content, v, replaceWith.(string))
			}
		}
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, err = emailer.construct(app, "user_expiry", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

// calls the send method in the underlying emailClient.
func (emailer *Emailer) send(email *Email, address ...string) error {
	return emailer.sender.send(emailer.fromName, emailer.fromAddr, email, address...)
}
