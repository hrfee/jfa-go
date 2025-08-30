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
	"maps"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	textTemplate "text/template"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/hrfee/jfa-go/easyproxy"
	lm "github.com/hrfee/jfa-go/logmessages"
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
	config             *Config
	storage            *Storage
	LoggerSet
}

// Message stores content.
type Message struct {
	Subject  string `json:"subject"`
	HTML     string `json:"html"`
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
}

func (emailer *Emailer) formatExpiry(expiry time.Time, tzaware bool) (d, t, expiresIn string) {
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
func NewEmailer(config *Config, storage *Storage, logs LoggerSet) *Emailer {
	emailer := &Emailer{
		fromAddr:  config.Section("email").Key("address").String(),
		fromName:  config.Section("email").Key("from").String(),
		lang:      storage.lang.Email[storage.lang.chosenEmailLang],
		LoggerSet: logs,
		config:    config,
		storage:   storage,
	}
	method := emailer.config.Section("email").Key("method").String()
	if method == "smtp" {
		enc := sMail.EncryptionSTARTTLS
		switch emailer.config.Section("smtp").Key("encryption").String() {
		case "ssl_tls":
			enc = sMail.EncryptionSSLTLS
		case "starttls":
			enc = sMail.EncryptionSTARTTLS
		case "none":
			enc = sMail.EncryptionNone
		}
		username := emailer.config.Section("smtp").Key("username").MustString("")
		password := emailer.config.Section("smtp").Key("password").String()
		if username == "" && password != "" {
			username = emailer.fromAddr
		}
		authType := sMail.AuthType(emailer.config.Section("smtp").Key("auth_type").MustInt(4))
		err := emailer.NewSMTP(emailer.config.Section("smtp").Key("server").String(), emailer.config.Section("smtp").Key("port").MustInt(465), username, password, enc, emailer.config.Section("smtp").Key("ssl_cert").MustString(""), emailer.config.Section("smtp").Key("hello_hostname").String(), emailer.config.Section("smtp").Key("cert_validation").MustBool(true), authType, emailer.config.proxyConfig)
		if err != nil {
			emailer.err.Printf(lm.FailedInitSMTP, err)
		}
	} else if method == "mailgun" {
		emailer.NewMailgun(emailer.config.Section("mailgun").Key("api_url").String(), emailer.config.Section("mailgun").Key("api_key").String(), emailer.config.proxyTransport)
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
func (emailer *Emailer) NewSMTP(server string, port int, username, password string, encryption sMail.Encryption, certPath string, helloHostname string, validateCertificate bool, authType sMail.AuthType, proxy *easyproxy.ProxyConfig) (err error) {
	sender := &SMTP{}
	sender.Client = sMail.NewSMTPClient()
	sender.Client.Encryption = encryption
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
			err = errors.New("failed to append cert to pool")
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
func (emailer *Emailer) NewMailgun(url, key string, transport *http.Transport) {
	sender := &Mailgun{
		client: mailgun.NewMailgun(strings.Split(emailer.fromAddr, "@")[1], key),
	}
	if transport != nil {
		cli := sender.client.Client()
		cli.Transport = transport
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

func (emailer *Emailer) construct(contentInfo CustomContentInfo, cc CustomContent, data map[string]any, msg *Message) error {
	if cc.Enabled {
		// Use template email, rather than the built-in's email file.
		contentInfo.SourceFile = customContent["TemplateEmail"].SourceFile
		content, err := templateEmail(cc.Content, contentInfo.Variables, contentInfo.Conditionals, data)
		if err != nil {
			emailer.err.Printf(lm.FailedConstructCustomContent, msg.Subject, err)
			return err
		}
		html := markdown.ToHTML([]byte(content), nil, markdownRenderer)
		text := stripMarkdown(content)
		templateData := map[string]interface{}{
			"text":      template.HTML(html),
			"plaintext": text,
			"md":        content,
		}
		if message, ok := data["message"]; ok {
			templateData["message"] = message
		}
		data = templateData
	}
	var err error = nil
	// Template the subject for bonus points
	if subject, err := templateEmail(msg.Subject, contentInfo.Variables, contentInfo.Conditionals, data); err == nil {
		msg.Subject = subject
	}

	var tpl templ
	msg.Text = ""
	msg.Markdown = ""
	msg.HTML = ""
	if substituteStrings == "" {
		data["jellyfin"] = "Jellyfin"
	} else {
		data["jellyfin"] = substituteStrings
	}
	var keys []string
	plaintext := emailer.config.Section("email").Key("plaintext").MustBool(false)
	if plaintext {
		if telegramEnabled || discordEnabled {
			keys = []string{"text"}
			msg.Text, msg.Markdown = "", ""
		} else {
			keys = []string{"text"}
			msg.Text = ""
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
			filesystem, fpath = emailer.config.GetPath(contentInfo.SourceFile.Section, contentInfo.SourceFile.SettingPrefix+"text")
		} else {
			filesystem, fpath = emailer.config.GetPath(contentInfo.SourceFile.Section, contentInfo.SourceFile.SettingPrefix+key)
		}
		if key == "html" {
			tpl, err = template.ParseFS(filesystem, fpath)
		} else {
			tpl, err = textTemplate.ParseFS(filesystem, fpath)
		}
		if err != nil {
			return fmt.Errorf("error reading from fs path \"%s\": %v", fpath, err)
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
			return err
		}
		if foundMarkdown {
			data["plaintext"], data["md"] = data["md"], data["plaintext"]
		}
		if key == "html" {
			msg.HTML = tplData.String()
		} else if key == "text" {
			msg.Text = tplData.String()
		} else {
			msg.Markdown = tplData.String()
		}
	}
	return nil
}

func (emailer *Emailer) baseValues(name string, username string, placeholders bool, values map[string]any) (CustomContentInfo, map[string]any, *Message) {
	contentInfo := customContent[name]
	template := map[string]any{
		"username": username,
		"message":  emailer.config.Section("messages").Key("message").String(),
	}
	maps.Copy(template, values)
	// When generating a version for the user to customise, we'll replace "variable" with "{variable}", so the templater used for custom content understands them.
	if placeholders {
		for _, v := range contentInfo.Variables {
			template[v] = "{" + v + "}"
		}
	}
	email := &Message{
		Subject: contentInfo.Subject(emailer.config, &emailer.lang),
	}
	return contentInfo, template, email
}

func (emailer *Emailer) constructConfirmation(code, username, key string, placeholders bool) (*Message, error) {
	if placeholders {
		username = "{username}"
	}
	contentInfo, template, msg := emailer.baseValues("EmailConfirmation", username, placeholders, map[string]any{
		"helloUser":     emailer.lang.Strings.template("helloUser", tmpl{"username": username}),
		"clickBelow":    emailer.lang.EmailConfirmation.get("clickBelow"),
		"ifItWasNotYou": emailer.lang.Strings.get("ifItWasNotYou"),
		"confirmEmail":  emailer.lang.EmailConfirmation.get("confirmEmail"),
	})
	if !placeholders {
		inviteLink := ExternalURI(nil)
		if code == "" { // Personal email change
			inviteLink = fmt.Sprintf("%s/my/confirm/%s", inviteLink, url.PathEscape(key))
		} else { // Invite email confirmation
			inviteLink = fmt.Sprintf("%s%s/%s?key=%s", inviteLink, PAGES.Form, code, url.PathEscape(key))
		}
		template["confirmationURL"] = inviteLink
	}
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructInvite(invite Invite, placeholders bool) (*Message, error) {
	expiry := invite.ValidTill
	d, t, expiresIn := emailer.formatExpiry(expiry, false)
	inviteLink := fmt.Sprintf("%s%s/%s", ExternalURI(nil), PAGES.Form, invite.Code)
	contentInfo, template, msg := emailer.baseValues("InviteEmail", "", placeholders, map[string]any{
		"hello":              emailer.lang.InviteEmail.get("hello"),
		"youHaveBeenInvited": emailer.lang.InviteEmail.get("youHaveBeenInvited"),
		"toJoin":             emailer.lang.InviteEmail.get("toJoin"),
		"linkButton":         emailer.lang.InviteEmail.get("linkButton"),
		"date":               d,
		"time":               t,
		"expiresInMinutes":   expiresIn,
		"inviteURL":          inviteLink,
		"inviteExpiry":       emailer.lang.InviteEmail.get("inviteExpiry"),
	})
	if !placeholders {
		template["inviteExpiry"] = emailer.lang.InviteEmail.template("inviteExpiry", template)
	}
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructExpiry(invite Invite, placeholders bool) (*Message, error) {
	expiry := formatDatetime(invite.ValidTill)
	contentInfo, template, msg := emailer.baseValues("InviteExpiry", "", placeholders, map[string]any{
		"inviteExpired":      emailer.lang.InviteExpiry.get("inviteExpired"),
		"notificationNotice": emailer.lang.InviteExpiry.get("notificationNotice"),
		"expiredAt":          emailer.lang.InviteExpiry.get("expiredAt"),
		"code":               "\"" + invite.Code + "\"",
		"time":               expiry,
	})
	if !placeholders {
		template["expiredAt"] = emailer.lang.InviteExpiry.template("expiredAt", template)
	}
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructCreated(username, address string, when time.Time, invite Invite, placeholders bool) (*Message, error) {
	// NOTE: This was previously invite.Created, not sure why.
	created := formatDatetime(when)
	contentInfo, template, msg := emailer.baseValues("UserCreated", username, placeholders, map[string]any{
		"aUserWasCreated":    emailer.lang.UserCreated.get("aUserWasCreated"),
		"nameString":         emailer.lang.Strings.get("name"),
		"addressString":      emailer.lang.Strings.get("emailAddress"),
		"timeString":         emailer.lang.UserCreated.get("time"),
		"notificationNotice": emailer.lang.UserCreated.get("notificationNotice"),
		"code":               "\"" + invite.Code + "\"",
		"name":               username,
		"time":               created,
		"address":            address,
	})
	if !placeholders {
		template["aUserWasCreated"] = emailer.lang.UserCreated.template("aUserWasCreated", template)
		if emailer.config.Section("email").Key("no_username").MustBool(false) {
			template["address"] = "n/a"
		}
	}
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructReset(pwr PasswordReset, placeholders bool) (*Message, error) {
	if placeholders {
		pwr.Username = "{username}"
	}
	d, t, expiresIn := emailer.formatExpiry(pwr.Expiry, true)
	linkResetEnabled := emailer.config.Section("password_resets").Key("link_reset").MustBool(false)
	contentInfo, template, msg := emailer.baseValues("PasswordReset", pwr.Username, placeholders, map[string]any{
		"helloUser":                emailer.lang.Strings.template("helloUser", tmpl{"username": pwr.Username}),
		"someoneHasRequestedReset": emailer.lang.PasswordReset.get("someoneHasRequestedReset"),
		"ifItWasYou":               emailer.lang.PasswordReset.get("ifItWasYou"),
		"ifItWasNotYou":            emailer.lang.Strings.get("ifItWasNotYou"),
		"pinString":                emailer.lang.PasswordReset.get("pin"),
		"codeExpiry":               emailer.lang.PasswordReset.get("codeExpiry"),
		"link_reset":               linkResetEnabled && !placeholders,
		"date":                     d,
		"time":                     t,
		"expiresInMinutes":         expiresIn,
		"pin":                      pwr.Pin,
	})
	if linkResetEnabled {
		template["ifItWasYou"] = emailer.lang.PasswordReset.get("ifItWasYouLink")
	}
	if !placeholders {
		template["codeExpiry"] = emailer.lang.PasswordReset.template("codeExpiry", template)
		if linkResetEnabled {
			pinLink, err := GenResetLink(pwr.Pin)
			if err != nil {
				template["link_reset"] = false
				emailer.info.Printf(lm.FailedGeneratePWRLink, err)
			} else {
				template["pin"] = pinLink
				// Only used in html email.
				template["pin_code"] = pwr.Pin
			}
		}
	}
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructDeleted(reason string, placeholders bool) (*Message, error) {
	if placeholders {
		reason = "{reason}"
	}
	contentInfo, template, msg := emailer.baseValues("UserDeleted", "", placeholders, map[string]any{
		"yourAccountWas": emailer.lang.UserDeleted.get("yourAccountWasDeleted"),
		"reasonString":   emailer.lang.Strings.get("reason"),
		"reason":         reason,
	})
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructDisabled(reason string, placeholders bool) (*Message, error) {
	if placeholders {
		reason = "{reason}"
	}
	contentInfo, template, msg := emailer.baseValues("UserDeleted", "", placeholders, map[string]any{
		"yourAccountWas": emailer.lang.UserDisabled.get("yourAccountWasDisabled"),
		"reasonString":   emailer.lang.Strings.get("reason"),
		"reason":         reason,
	})
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructEnabled(reason string, placeholders bool) (*Message, error) {
	if placeholders {
		reason = "{reason}"
	}
	contentInfo, template, msg := emailer.baseValues("UserDeleted", "", placeholders, map[string]any{
		"yourAccountWas": emailer.lang.UserEnabled.get("yourAccountWasEnabled"),
		"reasonString":   emailer.lang.Strings.get("reason"),
		"reason":         reason,
	})
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructExpiryAdjusted(username string, expiry time.Time, reason string, placeholders bool) (*Message, error) {
	if placeholders {
		username = "{username}"
	}
	exp := formatDatetime(expiry)
	contentInfo, template, msg := emailer.baseValues("UserExpiryAdjusted", username, placeholders, map[string]any{
		"helloUser":             emailer.lang.Strings.template("helloUser", tmpl{"username": username}),
		"yourExpiryWasAdjusted": emailer.lang.UserExpiryAdjusted.get("yourExpiryWasAdjusted"),
		"ifPreviouslyDisabled":  emailer.lang.UserExpiryAdjusted.get("ifPreviouslyDisabled"),
		"reasonString":          emailer.lang.Strings.get("reason"),
		"reason":                reason,
		"newExpiry":             exp,
	})
	cc := emailer.storage.MustGetCustomContentKey("UserExpiryAdjusted")
	if !placeholders {
		if !cc.Enabled {
			template["newExpiry"] = emailer.lang.UserExpiryAdjusted.template("newExpiry", tmpl{
				"date": exp,
			})
		}
	}
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructExpiryReminder(username string, expiry time.Time, placeholders bool) (*Message, error) {
	if placeholders {
		username = "{username}"
	}
	d, t, expiresIn := emailer.formatExpiry(expiry, false)
	contentInfo, template, msg := emailer.baseValues("ExpiryReminder", username, placeholders, map[string]any{
		"helloUser":                emailer.lang.Strings.template("helloUser", tmpl{"username": username}),
		"yourAccountIsDueToExpire": emailer.lang.ExpiryReminder.get("yourAccountIsDueToExpire"),
		"expiresIn":                expiresIn,
		"date":                     d,
		"time":                     t,
	})
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	if !placeholders {
		if !cc.Enabled && !expiry.IsZero() {
			template["yourAccountIsDueToExpire"] = emailer.lang.ExpiryReminder.template("yourAccountIsDueToExpire", template)
		}
	}
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructWelcome(username string, expiry time.Time, placeholders bool) (*Message, error) {
	var exp any = formatDatetime(expiry)
	if placeholders {
		username = "{username}"
		exp = "{yourAccountWillExpire}"
	}
	contentInfo, template, msg := emailer.baseValues("WelcomeEmail", username, placeholders, map[string]any{
		"welcome":           emailer.lang.WelcomeEmail.get("welcome"),
		"youCanLoginWith":   emailer.lang.WelcomeEmail.get("youCanLoginWith"),
		"jellyfinURLString": emailer.lang.WelcomeEmail.get("jellyfinURL"),
		"jellyfinURL":       emailer.config.Section("jellyfin").Key("public_server").String(),
		"usernameString":    emailer.lang.Strings.get("username"),
	})
	if !expiry.IsZero() || placeholders {
		template["yourAccountWillExpire"] = emailer.lang.WelcomeEmail.template("yourAccountWillExpire", tmpl{
			"date": exp,
		})
	}
	cc := emailer.storage.MustGetCustomContentKey("WelcomeEmail")
	if !placeholders {
		if cc.Enabled && !expiry.IsZero() {
			template["yourAccountWillExpire"] = exp
		}
	}
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
}

func (emailer *Emailer) constructUserExpired(placeholders bool) (*Message, error) {
	contentInfo, template, msg := emailer.baseValues("UserExpired", "", placeholders, map[string]any{
		"yourAccountHasExpired": emailer.lang.UserExpired.get("yourAccountHasExpired"),
		"contactTheAdmin":       emailer.lang.UserExpired.get("contactTheAdmin"),
	})
	cc := emailer.storage.MustGetCustomContentKey(contentInfo.Name)
	err := emailer.construct(contentInfo, cc, template, msg)
	return msg, err
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
	var err error = nil
	if matchUsername {
		user, err = app.jf.UserByName(address, false)
		if err == nil {
			ok = true
			return
		}
	}

	if matchEmail {
		emailAddresses := []EmailAddress{}
		err = app.storage.db.Find(&emailAddresses, badgerhold.Where("Addr").Eq(address))
		if err == nil && len(emailAddresses) > 0 {
			for _, emailUser := range emailAddresses {
				user, err = app.jf.UserByID(emailUser.JellyfinID, false)
				if err == nil {
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
				user, err = app.jf.UserByID(dcUser.JellyfinID, false)
				if err == nil {
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
				user, err = app.jf.UserByID(telegramUser.JellyfinID, false)
				if err == nil {
					ok = true
					return
				}
			}
		}
		matrixUsers := []MatrixUser{}
		err = app.storage.db.Find(&matrixUsers, badgerhold.Where("UserID").Eq(address))
		if err == nil && len(matrixUsers) > 0 {
			for _, matrixUser := range matrixUsers {
				user, err = app.jf.UserByID(matrixUser.JellyfinID, false)
				if err == nil {
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
