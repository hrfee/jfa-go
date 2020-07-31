package main

import (
	"bytes"
	// "context"
	"context"
	"fmt"
	"github.com/knz/strtime"
	"github.com/mailgun/mailgun-go/v4"
	"html/template"
	"net/smtp"
	"strings"
	"time"
)

type Emailer struct {
	smtpAuth                                 smtp.Auth
	sendType, sendMethod, fromAddr, fromName string
	content                                  Email
	mg                                       *mailgun.MailgunImpl
}

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

func (email *Emailer) init(ctx *appContext) {
	email.fromAddr = ctx.config.Section("email").Key("address").String()
	email.fromName = ctx.config.Section("email").Key("from").String()
	email.sendMethod = ctx.config.Section("email").Key("method").String()
	if email.sendMethod == "mailgun" {
		email.mg = mailgun.NewMailgun(strings.Split(email.fromAddr, "@")[1], ctx.config.Section("mailgun").Key("api_key").String())
		api_url := ctx.config.Section("mailgun").Key("api_url").String()
		if strings.Contains(api_url, "messages") {
			api_url = api_url[0:strings.LastIndex(api_url, "/")]
			api_url = api_url[0:strings.LastIndex(api_url, "/")]
		}
		email.mg.SetAPIBase(api_url)
	}
}

func (email *Emailer) constructInvite(code string, invite Invite, ctx *appContext) error {
	email.content.subject = ctx.config.Section("invite_emails").Key("subject").String()
	expiry := invite.ValidTill
	d, t, expires_in := email.formatExpiry(expiry, false, ctx.datePattern, ctx.timePattern)
	message := ctx.config.Section("email").Key("message").String()
	invite_link := ctx.config.Section("invite_emails").Key("url_base").String()
	invite_link = fmt.Sprintf("%s/%s", invite_link, code)

	for _, key := range []string{"html", "text"} {
		fpath := ctx.config.Section("invite_emails").Key("email_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			fmt.Println("failed email", err)
			return err
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
			fmt.Println("failed email", err)
			return err
		}
		if key == "html" {
			email.content.html = tplData.String()
		} else {
			email.content.text = tplData.String()
		}
	}
	email.sendType = "invite"
	return nil
}

func (email *Emailer) constructExpiry(code string, invite Invite, ctx *appContext) error {
	email.content.subject = "Notice: Invite expired"
	expiry := ctx.formatDatetime(invite.ValidTill)
	for _, key := range []string{"html", "text"} {
		fpath := ctx.config.Section("notifications").Key("expiry_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			fmt.Println("failed email", err)
			return err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"code":   code,
			"expiry": expiry,
		})
		if err != nil {
			fmt.Println("failed email", err)
			return err
		}
		if key == "html" {
			email.content.html = tplData.String()
		} else {
			email.content.text = tplData.String()
		}
	}
	email.sendType = "expiry"
	return nil
}

func (email *Emailer) constructCreated(code, username, address string, invite Invite, ctx *appContext) error {
	email.content.subject = "Notice: User created"
	created := ctx.formatDatetime(invite.Created)
	var tplAddress string
	if ctx.config.Section("email").Key("no_username").MustBool(false) {
		tplAddress = "n/a"
	} else {
		tplAddress = address
	}
	for _, key := range []string{"html", "text"} {
		fpath := ctx.config.Section("notifications").Key("created_" + key).String()
		tpl, err := template.ParseFiles(fpath)
		if err != nil {
			fmt.Println("failed email", err)
			return err
		}
		var tplData bytes.Buffer
		err = tpl.Execute(&tplData, map[string]string{
			"code":     code,
			"username": username,
			"address":  tplAddress,
			"time":     created,
		})
		if err != nil {
			fmt.Println("failed email", err)
			return err
		}
		if key == "html" {
			email.content.html = tplData.String()
		} else {
			email.content.text = tplData.String()
		}
	}
	email.sendType = "created"
	return nil
}

func (email *Emailer) send(address string, ctx *appContext) error {
	if email.sendMethod == "mailgun" {
		// reqData := map[string]string{
		// 	"to":      fmt.Sprintf("%s <%s>", "test", email.to),
		// 	"from":    email.fromAddr,
		// 	"subject": email.subject,
		// }
		// if email.sendType == "invite" {
		// 	reqData["text"] = email.invite.text
		// 	reqData["html"] = email.invite.html
		// }
		// data := &bytes.Buffer{}
		// encoder := json.NewEncoder(data)
		// encoder.SetEscapeHTML(false)
		// err := encoder.Encode(reqData)
		// fmt.Println("marshaled:", data)
		// if err != nil {
		// 	fmt.Println("Failed marshal:", err, ">", data)
		// 	return err
		// }
		// var req *http.Request
		// req, err = http.NewRequest("POST", ctx.config.Section("mailgun").Key("api_url").String(), data)
		// req.SetBasicAuth("api", ctx.config.Section("mailgun").Key("api_key").String())
		// var resp *http.Response
		// resp, err = email.httpClient.Do(req)
		// if err != nil || !(resp.StatusCode == 200 || resp.StatusCode == 204) {
		// 	fmt.Println("failed send:", err, resp.StatusCode)
		// 	fmt.Println("resp:", resp.Header.Get("Content-Encoding"), resp.Body)
		// }
		message := email.mg.NewMessage(
			fmt.Sprintf("%s <%s>", email.fromName, email.fromAddr),
			email.content.subject,
			email.content.text,
			address)
		message.SetHtml(email.content.html)
		mgctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		_, id, err := email.mg.Send(mgctx, message)
		fmt.Println("mailgun:", id, err)

	}
	return nil
}
