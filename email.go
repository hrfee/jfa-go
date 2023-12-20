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
	"io/fs"
	"net/url"
	"os"
	"strconv"
	"strings"
	textTemplate "text/template"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/hrfee/jfa-go/easyproxy"
	"github.com/hrfee/mediabrowser"
	"github.com/itchyny/timefmt-go"
	"github.com/mailgun/mailgun-go/v4"
	"github.com/timshannon/badgerhold/v4"
	sMail "github.com/xhit/go-simple-mail/v2"
)

var markdownRenderer = html.NewRenderer(html.RendererOptions{Flags: html.Smartypants})

// EmailClient implements email sending, right now via smtp, mailgun or a dummy client.
type EmailClient interface {
	Send(fromName, fromAddr string, message *Message, address ...string) error
}

// Emailer contains the email sender, translations, and methods to construct messages.
type Emailer struct {
	fromAddr, fromName string
	lang               emailLang
	sender             EmailClient
}

// Message stores content.
type Message struct {
	Subject  string `json:"subject"`
	HTML     string `json:"html"`
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
}

func (emailer *Emailer) formatExpiry(expiry time.Time, tzaware bool, datePattern, timePattern string) (d, t, expiresIn string) {
	d = timefmt.Format(expiry, datePattern)
	t = timefmt.Format(expiry, timePattern)
	currentTime := time.Now()
	if tzaware {
		currentTime = currentTime.UTC()
	}
	_, _, days, hours, minutes, _ := timeDiff(expiry, currentTime)
	if days != 0 {
		expiresIn += strconv.Itoa(days) + "d "
	}
	if hours != 0 {
		expiresIn += strconv.Itoa(hours) + "h "
	}
	if minutes != 0 {
		expiresIn += strconv.Itoa(minutes) + "m "
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
		sslTLS := false
		if app.config.Section("smtp").Key("encryption").String() == "ssl_tls" {
			sslTLS = true
		}
		username := app.config.Section("smtp").Key("username").MustString("")
		password := app.config.Section("smtp").Key("password").String()
		if username == "" && password != "" {
			username = emailer.fromAddr
		}
		var proxyConf *easyproxy.ProxyConfig = nil
		if app.proxyEnabled {
			proxyConf = &app.proxyConfig
		}
		authType := sMail.AuthType(app.config.Section("smtp").Key("auth_type").MustInt(4))
		err := emailer.NewSMTP(app.config.Section("smtp").Key("server").String(), app.config.Section("smtp").Key("port").MustInt(465), username, password, sslTLS, app.config.Section("smtp").Key("ssl_cert").MustString(""), app.config.Section("smtp").Key("hello_hostname").String(), app.config.Section("smtp").Key("cert_validation").MustBool(true), authType, proxyConf)
		if err != nil {
			app.err.Printf("Error while initiating SMTP mailer: %v", err)
		}
	} else if method == "mailgun" {
		emailer.NewMailgun(app.config.Section("mailgun").Key("api_url").String(), app.config.Section("mailgun").Key("api_key").String())
	} else if method == "dummy" {
		emailer.sender = &DummyClient{}
	}
	return emailer
}

// DummyClient just logs the email to the console for debugging purposes. It can be used by settings [email]/method to "dummy".
type DummyClient struct{}

func (dc *DummyClient) Send(fromName, fromAddr string, email *Message, address ...string) error {
	fmt.Printf("FROM: %s <%s>\nTO: %s\nTEXT: %s\n", fromName, fromAddr, strings.Join(address, ", "), email.Text)
	return nil
}

// SMTP supports SSL/TLS and STARTTLS; implements EmailClient.
type SMTP struct {
	Client *sMail.SMTPServer
}

// NewSMTP returns an SMTP emailClient.
func (emailer *Emailer) NewSMTP(server string, port int, username, password string, sslTLS bool, certPath string, helloHostname string, validateCertificate bool, authType sMail.AuthType, proxy *easyproxy.ProxyConfig) (err error) {
	sender := &SMTP{}
	sender.Client = sMail.NewSMTPClient()
	if sslTLS {
		sender.Client.Encryption = sMail.EncryptionSSLTLS
	} else {
		sender.Client.Encryption = sMail.EncryptionSTARTTLS
	}
	if username != "" || password != "" {
		sender.Client.Authentication = authType
		sender.Client.Username = username
		sender.Client.Password = password
	}
	sender.Client.Helo = helloHostname
	sender.Client.ConnectTimeout, sender.Client.SendTimeout = 15*time.Second, 15*time.Second
	sender.Client.Host = server
	sender.Client.Port = port
	sender.Client.KeepAlive = false

	// x509.SystemCertPool is unavailable on windows
	if PLATFORM == "windows" {
		sender.Client.TLSConfig = &tls.Config{
			InsecureSkipVerify: !validateCertificate,
			ServerName:         server,
		}
		if proxy != nil {
			sender.Client.CustomConn, err = easyproxy.NewConn(*proxy, fmt.Sprintf("%s:%d", server, port), sender.Client.TLSConfig)
		}
		emailer.sender = sender
		return
	}
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
	sender.Client.TLSConfig = &tls.Config{
		InsecureSkipVerify: !validateCertificate,
		ServerName:         server,
		RootCAs:            rootCAs,
	}
	if proxy != nil {
		sender.Client.CustomConn, err = easyproxy.NewConn(*proxy, fmt.Sprintf("%s:%d", server, port), sender.Client.TLSConfig)
	}
	emailer.sender = sender
	return
}

func (sm *SMTP) Send(fromName, fromAddr string, email *Message, address ...string) error {
	from := fmt.Sprintf("%s <%s>", fromName, fromAddr)
	var cli *sMail.SMTPClient
	var err error
	cli, err = sm.Client.Connect()
	if err != nil {
		return err
	}
	defer cli.Close()
	e := sMail.NewMSG()
	e.SetFrom(from)
	e.SetSubject(email.Subject)
	e.AddTo(address...)
	e.SetBody(sMail.TextPlain, email.Text)
	if email.HTML != "" {
		e.AddAlternative(sMail.TextHTML, email.HTML)
	}
	err = e.Send(cli)
	return err
}

// Mailgun client implements EmailClient.
type Mailgun struct {
	client *mailgun.MailgunImpl
}

// NewMailgun returns a Mailgun emailClient.
func (emailer *Emailer) NewMailgun(url, key string) {
	sender := &Mailgun{
		client: mailgun.NewMailgun(strings.Split(emailer.fromAddr, "@")[1], key),
	}
	// Mailgun client takes the base url, so we need to trim off the end (e.g 'v3/messages')
	if strings.Contains(url, "messages") {
		url = url[0:strings.LastIndex(url, "/")]
	}
	if strings.Contains(url, "v3") {
		url = url[0:strings.LastIndex(url, "/")]
	}
	sender.client.SetAPIBase(url)
	emailer.sender = sender
}

func (mg *Mailgun) Send(fromName, fromAddr string, email *Message, address ...string) error {
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

type templ interface {
	Execute(wr io.Writer, data interface{}) error
}

func (emailer *Emailer) construct(app *appContext, section, keyFragment string, data map[string]interface{}) (html, text, markdown string, err error) {
	var tpl templ
	if substituteStrings == "" {
		data["jellyfin"] = "Jellyfin"
	} else {
		data["jellyfin"] = substituteStrings
	}
	var keys []string
	plaintext := app.config.Section("email").Key("plaintext").MustBool(false)
	if plaintext {
		if telegramEnabled || discordEnabled {
			keys = []string{"text"}
			text, markdown = "", ""
		} else {
			keys = []string{"text"}
			text = ""
		}
	} else {
		if telegramEnabled || discordEnabled {
			keys = []string{"html", "text", "markdown"}
		} else {
			keys = []string{"html", "text"}
		}
	}
	for _, key := range keys {
		var filesystem fs.FS
		var fpath string
		if key == "markdown" {
			filesystem, fpath = app.GetPath(section, keyFragment+"text")
		} else {
			filesystem, fpath = app.GetPath(section, keyFragment+key)
		}
		if key == "html" {
			tpl, err = template.ParseFS(filesystem, fpath)
		} else {
			tpl, err = textTemplate.ParseFS(filesystem, fpath)
		}
		if err != nil {
			return
		}
		// For constructTemplate, if "md" is found in data it's used in stead of "text".
		foundMarkdown := false
		if key == "markdown" {
			_, foundMarkdown = data["md"]
			if foundMarkdown {
				data["plaintext"], data["md"] = data["md"], data["plaintext"]
			}
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, data)
		if err != nil {
			return
		}
		if foundMarkdown {
			data["plaintext"], data["md"] = data["md"], data["plaintext"]
		}
		if key == "html" {
			html = tplData.String()
		} else if key == "text" {
			text = tplData.String()
		} else {
			markdown = tplData.String()
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
		message := app.config.Section("messages").Key("message").String()
		inviteLink := app.config.Section("invite_emails").Key("url_base").String()
		if code == "" { // Personal email change
			if strings.HasSuffix(inviteLink, "/invite") {
				inviteLink = strings.TrimSuffix(inviteLink, "/invite")
			}
			inviteLink = fmt.Sprintf("%s/my/confirm/%s", inviteLink, url.PathEscape(key))
		} else { // Invite email confirmation
			if !strings.HasSuffix(inviteLink, "/invite") {
				inviteLink += "/invite"
			}
			inviteLink = fmt.Sprintf("%s/%s?key=%s", inviteLink, code, url.PathEscape(key))
		}
		template["helloUser"] = emailer.lang.Strings.template("helloUser", tmpl{"username": username})
		template["confirmationURL"] = inviteLink
		template["message"] = message
	}
	return template
}

func (emailer *Emailer) constructConfirmation(code, username, key string, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("email_confirmation").Key("subject").MustString(emailer.lang.EmailConfirmation.get("title")),
	}
	var err error
	template := emailer.confirmationValues(code, username, key, app, noSub)
	message := app.storage.MustGetCustomContentKey("EmailConfirmation")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "email_confirmation", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

// username is optional, but should only be passed once.
func (emailer *Emailer) constructTemplate(subject, md string, app *appContext, username ...string) (*Message, error) {
	if len(username) != 0 {
		md = templateEmail(md, []string{"{username}"}, nil, map[string]interface{}{"username": username[0]})
		subject = templateEmail(subject, []string{"{username}"}, nil, map[string]interface{}{"username": username[0]})
	}
	email := &Message{Subject: subject}
	html := markdown.ToHTML([]byte(md), nil, markdownRenderer)
	text := stripMarkdown(md)
	message := app.config.Section("messages").Key("message").String()
	var err error
	data := map[string]interface{}{
		"text":      template.HTML(html),
		"plaintext": text,
		"message":   message,
		"md":        md,
	}
	if len(username) != 0 {
		data["username"] = username[0]
	}
	email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "template_email", "email_", data)
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) inviteValues(code string, invite Invite, app *appContext, noSub bool) map[string]interface{} {
	expiry := invite.ValidTill
	d, t, expiresIn := emailer.formatExpiry(expiry, false, app.datePattern, app.timePattern)
	message := app.config.Section("messages").Key("message").String()
	inviteLink := app.config.Section("invite_emails").Key("url_base").String()
	if !strings.HasSuffix(inviteLink, "/invite") {
		inviteLink += "/invite"
	}
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

func (emailer *Emailer) constructInvite(code string, invite Invite, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("invite_emails").Key("subject").MustString(emailer.lang.InviteEmail.get("title")),
	}
	template := emailer.inviteValues(code, invite, app, noSub)
	var err error
	message := app.storage.MustGetCustomContentKey("InviteEmail")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "invite_emails", "email_", template)
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

func (emailer *Emailer) constructExpiry(code string, invite Invite, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: emailer.lang.InviteExpiry.get("title"),
	}
	var err error
	template := emailer.expiryValues(code, invite, app, noSub)
	message := app.storage.MustGetCustomContentKey("InviteExpiry")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "notifications", "expiry_", template)
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

func (emailer *Emailer) constructCreated(code, username, address string, invite Invite, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: emailer.lang.UserCreated.get("title"),
	}
	template := emailer.createdValues(code, username, address, invite, app, noSub)
	var err error
	message := app.storage.MustGetCustomContentKey("UserCreated")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "notifications", "created_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) resetValues(pwr PasswordReset, app *appContext, noSub bool) map[string]interface{} {
	d, t, expiresIn := emailer.formatExpiry(pwr.Expiry, true, app.datePattern, app.timePattern)
	message := app.config.Section("messages").Key("message").String()
	template := map[string]interface{}{
		"someoneHasRequestedReset": emailer.lang.PasswordReset.get("someoneHasRequestedReset"),
		"ifItWasNotYou":            emailer.lang.Strings.get("ifItWasNotYou"),
		"pinString":                emailer.lang.PasswordReset.get("pin"),
		"link_reset":               false,
		"message":                  "",
		"username":                 pwr.Username,
		"date":                     d,
		"time":                     t,
		"expiresInMinutes":         expiresIn,
	}
	linkResetEnabled := app.config.Section("password_resets").Key("link_reset").MustBool(false)
	if linkResetEnabled {
		template["ifItWasYou"] = emailer.lang.PasswordReset.get("ifItWasYouLink")
	} else {
		template["ifItWasYou"] = emailer.lang.PasswordReset.get("ifItWasYou")
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
		if linkResetEnabled {
			pinLink, err := app.GenResetLink(pwr.Pin)
			if err == nil {
				// Strip /invite form end of this URL, ik its ugly.
				template["link_reset"] = true
				template["pin"] = pinLink
				// Only used in html email.
				template["pin_code"] = pwr.Pin
			} else {
				app.info.Println("Couldn't generate PWR link: %v", err)
				template["pin"] = pwr.Pin
			}
		} else {
			template["pin"] = pwr.Pin
		}
		template["message"] = message
	}
	return template
}

func (emailer *Emailer) constructReset(pwr PasswordReset, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("password_resets").Key("subject").MustString(emailer.lang.PasswordReset.get("title")),
	}
	template := emailer.resetValues(pwr, app, noSub)
	var err error
	message := app.storage.MustGetCustomContentKey("PasswordReset")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "password_resets", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) deletedValues(reason string, app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"yourAccountWas": emailer.lang.UserDeleted.get("yourAccountWasDeleted"),
		"reasonString":   emailer.lang.Strings.get("reason"),
		"message":        "",
	}
	if noSub {
		empty := []string{"reason"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["reason"] = reason
		template["message"] = app.config.Section("messages").Key("message").String()
	}
	return template
}

func (emailer *Emailer) constructDeleted(reason string, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("deletion").Key("subject").MustString(emailer.lang.UserDeleted.get("title")),
	}
	var err error
	template := emailer.deletedValues(reason, app, noSub)
	message := app.storage.MustGetCustomContentKey("UserDeleted")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "deletion", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) disabledValues(reason string, app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"yourAccountWas": emailer.lang.UserDisabled.get("yourAccountWasDisabled"),
		"reasonString":   emailer.lang.Strings.get("reason"),
		"message":        "",
	}
	if noSub {
		empty := []string{"reason"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["reason"] = reason
		template["message"] = app.config.Section("messages").Key("message").String()
	}
	return template
}

func (emailer *Emailer) constructDisabled(reason string, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("disable_enable").Key("subject_disabled").MustString(emailer.lang.UserDisabled.get("title")),
	}
	var err error
	template := emailer.disabledValues(reason, app, noSub)
	message := app.storage.MustGetCustomContentKey("UserDisabled")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "disable_enable", "disabled_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) enabledValues(reason string, app *appContext, noSub bool) map[string]interface{} {
	template := map[string]interface{}{
		"yourAccountWas": emailer.lang.UserEnabled.get("yourAccountWasEnabled"),
		"reasonString":   emailer.lang.Strings.get("reason"),
		"message":        "",
	}
	if noSub {
		empty := []string{"reason"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["reason"] = reason
		template["message"] = app.config.Section("messages").Key("message").String()
	}
	return template
}

func (emailer *Emailer) constructEnabled(reason string, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("disable_enable").Key("subject_enabled").MustString(emailer.lang.UserEnabled.get("title")),
	}
	var err error
	template := emailer.enabledValues(reason, app, noSub)
	message := app.storage.MustGetCustomContentKey("UserEnabled")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "disable_enable", "enabled_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

func (emailer *Emailer) welcomeValues(username string, expiry time.Time, app *appContext, noSub bool, custom bool) map[string]interface{} {
	template := map[string]interface{}{
		"welcome":               emailer.lang.WelcomeEmail.get("welcome"),
		"youCanLoginWith":       emailer.lang.WelcomeEmail.get("youCanLoginWith"),
		"jellyfinURLString":     emailer.lang.WelcomeEmail.get("jellyfinURL"),
		"usernameString":        emailer.lang.Strings.get("username"),
		"message":               "",
		"yourAccountWillExpire": "",
	}
	if noSub {
		empty := []string{"jellyfinURL", "username", "yourAccountWillExpire"}
		for _, v := range empty {
			template[v] = "{" + v + "}"
		}
	} else {
		template["jellyfinURL"] = app.config.Section("jellyfin").Key("public_server").String()
		template["username"] = username
		template["message"] = app.config.Section("messages").Key("message").String()
		exp := app.formatDatetime(expiry)
		if !expiry.IsZero() {
			if custom {
				template["yourAccountWillExpire"] = exp
			} else if !expiry.IsZero() {
				template["yourAccountWillExpire"] = emailer.lang.WelcomeEmail.template("yourAccountWillExpire", tmpl{
					"date": exp,
				})
			}
		}
	}
	return template
}

func (emailer *Emailer) constructWelcome(username string, expiry time.Time, app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("welcome_email").Key("subject").MustString(emailer.lang.WelcomeEmail.get("title")),
	}
	var err error
	var template map[string]interface{}
	message := app.storage.MustGetCustomContentKey("WelcomeEmail")
	if message.Enabled {
		template = emailer.welcomeValues(username, expiry, app, noSub, true)
	} else {
		template = emailer.welcomeValues(username, expiry, app, noSub, false)
	}
	if noSub {
		template["yourAccountWillExpire"] = emailer.lang.WelcomeEmail.template("yourAccountWillExpire", tmpl{
			"date": "{yourAccountWillExpire}",
		})
	}
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			message.Conditionals,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "welcome_email", "email_", template)
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
		template["message"] = app.config.Section("messages").Key("message").String()
	}
	return template
}

func (emailer *Emailer) constructUserExpired(app *appContext, noSub bool) (*Message, error) {
	email := &Message{
		Subject: app.config.Section("user_expiry").Key("subject").MustString(emailer.lang.UserExpired.get("title")),
	}
	var err error
	template := emailer.userExpiredValues(app, noSub)
	message := app.storage.MustGetCustomContentKey("UserExpired")
	if message.Enabled {
		content := templateEmail(
			message.Content,
			message.Variables,
			nil,
			template,
		)
		email, err = emailer.constructTemplate(email.Subject, content, app)
	} else {
		email.HTML, email.Text, email.Markdown, err = emailer.construct(app, "user_expiry", "email_", template)
	}
	if err != nil {
		return nil, err
	}
	return email, nil
}

// calls the send method in the underlying emailClient.
func (emailer *Emailer) send(email *Message, address ...string) error {
	return emailer.sender.Send(emailer.fromName, emailer.fromAddr, email, address...)
}

func (app *appContext) sendByID(email *Message, ID ...string) (err error) {
	for _, id := range ID {
		if tgChat, ok := app.storage.GetTelegramKey(id); ok && tgChat.Contact && telegramEnabled {
			err = app.telegram.Send(email, tgChat.ChatID)
			// if err != nil {
			// 	return err
			// }
		}
		if dcChat, ok := app.storage.GetDiscordKey(id); ok && dcChat.Contact && discordEnabled {
			err = app.discord.Send(email, dcChat.ChannelID)
			// if err != nil {
			// 	return err
			// }
		}
		if mxChat, ok := app.storage.GetMatrixKey(id); ok && mxChat.Contact && matrixEnabled {
			err = app.matrix.Send(email, mxChat)
			// if err != nil {
			// 	return err
			// }
		}
		if address, ok := app.storage.GetEmailsKey(id); ok && address.Contact && emailEnabled {
			err = app.email.send(email, address.Addr)
			// if err != nil {
			// 	return err
			// }
		}
		// if err != nil {
		// 	return err
		// }
	}
	return
}

func (app *appContext) getAddressOrName(jfID string) string {
	if dcChat, ok := app.storage.GetDiscordKey(jfID); ok && dcChat.Contact && discordEnabled {
		return RenderDiscordUsername(dcChat)
	}
	if tgChat, ok := app.storage.GetTelegramKey(jfID); ok && tgChat.Contact && telegramEnabled {
		return "@" + tgChat.Username
	}
	if addr, ok := app.storage.GetEmailsKey(jfID); ok {
		return addr.Addr
	}
	if mxChat, ok := app.storage.GetMatrixKey(jfID); ok && mxChat.Contact && matrixEnabled {
		return mxChat.UserID
	}
	return ""
}

// ReverseUserSearch returns the jellyfin ID of the user with the given username, email, or contact method username.
// returns "" if none found. returns only the first match, might be an issue if there are users with the same contact method usernames.
func (app *appContext) ReverseUserSearch(address string, matchUsername, matchEmail, matchContactMethod bool) (user mediabrowser.User, ok bool) {
	ok = false
	var status int
	var err error = nil
	if matchUsername {
		user, status, err = app.jf.UserByName(address, false)
		if status == 200 && err == nil {
			ok = true
			return
		}
	}

	if matchEmail {
		emailAddresses := []EmailAddress{}
		err = app.storage.db.Find(&emailAddresses, badgerhold.Where("Addr").Eq(address))
		if err == nil && len(emailAddresses) > 0 {
			for _, emailUser := range emailAddresses {
				user, status, err = app.jf.UserByID(emailUser.JellyfinID, false)
				if status == 200 && err == nil {
					ok = true
					return
				}
			}
		}
	}

	// Dont know how we'd use badgerhold when we need to render each username,
	// Apart from storing the rendered name in the db.
	if matchContactMethod {
		for _, dcUser := range app.storage.GetDiscord() {
			if RenderDiscordUsername(dcUser) == strings.ToLower(address) {
				user, status, err = app.jf.UserByID(dcUser.JellyfinID, false)
				if status == 200 && err == nil {
					ok = true
					return
				}
			}
		}
		tgUsername := strings.TrimPrefix(address, "@")
		telegramUsers := []TelegramUser{}
		err = app.storage.db.Find(&telegramUsers, badgerhold.Where("Username").Eq(tgUsername))
		if err == nil && len(telegramUsers) > 0 {
			for _, telegramUser := range telegramUsers {
				user, status, err = app.jf.UserByID(telegramUser.JellyfinID, false)
				if status == 200 && err == nil {
					ok = true
					return
				}
			}
		}
		matrixUsers := []MatrixUser{}
		err = app.storage.db.Find(&matrixUsers, badgerhold.Where("UserID").Eq(address))
		if err == nil && len(matrixUsers) > 0 {
			for _, matrixUser := range matrixUsers {
				user, status, err = app.jf.UserByID(matrixUser.JellyfinID, false)
				if status == 200 && err == nil {
					ok = true
					return
				}
			}
		}
	}
	return
}

// EmailAddressExists returns whether or not a user with the given email address exists.
func (app *appContext) EmailAddressExists(address string) bool {
	c, err := app.storage.db.Count(&EmailAddress{}, badgerhold.Where("Addr").Eq(address))
	return err != nil || c > 0
}
