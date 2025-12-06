package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/hrfee/jfa-go/logger"
	"github.com/lithammer/shortuuid/v3"
	"github.com/timshannon/badgerhold/v4"
)

var db *badgerhold.Store

func dbClose(e *Emailer) {
	e.storage.db.Close()
	e.storage.db = nil
	db = nil
}

func Fatal(err any) {
	fmt.Printf("Fatal log function called: %+v\n", err)
}

// NewTestEmailer initialises most of what the emailer depends on, which happens to be most of the app.
func NewTestEmailer() (*Emailer, error) {
	emailer := &Emailer{
		fromAddr: "from@addr",
		fromName: "fromName",
		LoggerSet: LoggerSet{
			info:  logger.NewLogger(os.Stdout, "[TEST INFO] ", log.Ltime, color.FgHiWhite),
			err:   logger.NewLogger(os.Stdout, "[TEST ERROR] ", log.Ltime|log.Lshortfile, color.FgRed),
			debug: logger.NewLogger(os.Stdout, "[TEST DEBUG] ", log.Ltime|log.Lshortfile, color.FgYellow),
		},
		sender: &DummyClient{},
	}
	// Assume our working directory is the root of the repo
	wd, _ := os.Getwd()
	loadFilesystems(filepath.Join(wd, "build"), logger.NewEmptyLogger())
	dConfig, err := fs.ReadFile(localFS, "config-default.ini")
	if err != nil {
		return emailer, err
	}

	// Force emailer to construct markdown
	discordEnabled = true
	noInfoLS := emailer.LoggerSet
	noInfoLS.info = logger.NewEmptyLogger()
	emailer.config, err = NewConfig(dConfig, "/tmp/jfa-go-test", noInfoLS)
	if err != nil {
		return emailer, err
	}
	emailer.storage = NewStorage("/tmp/db", emailer.debug, func(k string) DebugLogAction { return LogAll })
	emailer.storage.loadLang(langFS)

	emailer.storage.lang.chosenAdminLang = emailer.config.Section("ui").Key("language-admin").MustString("en-us")
	emailer.storage.lang.chosenEmailLang = emailer.config.Section("email").Key("language").MustString("en-us")
	emailer.storage.lang.chosenPWRLang = emailer.config.Section("password_resets").Key("language").MustString("en-us")
	emailer.storage.lang.chosenTelegramLang = emailer.config.Section("telegram").Key("language").MustString("en-us")

	opts := badgerhold.DefaultOptions
	opts.Dir = "/tmp/jfa-go-test-db"
	opts.ValueDir = opts.Dir
	opts.SyncWrites = false
	opts.Logger = nil
	emailer.storage.db, err = badgerhold.Open(opts)
	// emailer.info.Printf("DB Opened")
	db = emailer.storage.db
	if err != nil {
		return emailer, err
	}

	emailer.lang = emailer.storage.lang.Email[emailer.storage.lang.chosenEmailLang]
	emailer.info.SetFatalFunc(Fatal)
	emailer.err.SetFatalFunc(Fatal)
	return emailer, err
}

func testDummyEmailerInit(t *testing.T) *Emailer {
	e, err := NewTestEmailer()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	return e
}

func TestDummyEmailerInit(t *testing.T) {
	dbClose(testDummyEmailerInit(t))
}

func testContent(e *Emailer, cci CustomContentInfo, t *testing.T, testFunc func(t *testing.T)) {
	e.storage.DeleteCustomContentKey(cci.Name)
	t.Run(cci.Name, testFunc)
	cc := CustomContent{
		Name:    cci.Name,
		Enabled: true,
	}
	cc.Content = "start test content "
	for _, v := range cci.Variables {
		cc.Content += "{" + v + "}"
	}
	cc.Content += " end test content"
	e.storage.SetCustomContentKey(cci.Name, cc)
	t.Run(cci.Name+" Custom", testFunc)
	e.storage.DeleteCustomContentKey(cci.Name)
}

// constructConfirmation(code, username, key string, placeholders bool)
func TestConfirmation(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	// non-blank key, link should therefore not be a /my/confirm one
	if db == nil {
		t.Fatalf("db nil")
	}
	testContent(e, customContent["EmailConfirmation"], t, func(t *testing.T) {
		code := shortuuid.New()
		username := shortuuid.New()
		key := shortuuid.New()
		msg, err := e.constructConfirmation(code, username, key, false)
		t.Run("FromInvite", func(t *testing.T) {
			if err != nil {
				t.Fatalf("failed construct: %+v", err)
			}
			for _, content := range []string{msg.Text, msg.HTML} {
				if strings.Contains(content, "/my/confirm") {
					t.Fatalf("/my/confirm link generated instead of invite confirm link: %s", content)
				}
				if !strings.Contains(content, code) {
					t.Fatalf("code not found in output: %s", content)
				}
				if !strings.Contains(content, key) {
					t.Fatalf("key not found in output: %s", content)
				}
				if !strings.Contains(content, username) {
					t.Fatalf("username not found in output: %s", content)
				}
			}
		})
		code = ""
		msg, err = e.constructConfirmation(code, username, key, false)
		t.Run("FromMyAccount", func(t *testing.T) {
			if err != nil {
				t.Fatalf("failed construct: %+v", err)
			}
			for _, content := range []string{msg.Text, msg.HTML} {
				if !strings.Contains(content, "/my/confirm") {
					t.Fatalf("/my/confirm link not generated: %s", content)
				}
				if !strings.Contains(content, key) {
					t.Fatalf("key not found in output: %s", content)
				}
				if !strings.Contains(content, username) {
					t.Fatalf("username not found in output: %s", content)
				}
			}
		})
	})
}

// constructInvite(invite Invite, placeholders bool)
func TestInvite(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	// Fix date/time format
	datePattern = "%d/%m/%y"
	timePattern = "%H:%M"
	testContent(e, customContent["InviteEmail"], t, func(t *testing.T) {
		inv := Invite{
			Code:      shortuuid.New(),
			Created:   time.Now(),
			ValidTill: time.Now().Add(30 * time.Minute),
		}
		msg, err := e.constructInvite(&inv, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, inv.Code) {
				t.Fatalf("code not found in output: %s", content)
			}
			if !strings.Contains(content, "30m") {
				t.Fatalf("expiry not found in output: %s", content)
			}
		}
	})
}

// constructExpiry(code string, invite Invite, placeholders bool)
func TestExpiry(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	// Fix date/time format
	datePattern = "%d/%m/%y"
	timePattern = "%H:%M"
	testContent(e, customContent["InviteExpiry"], t, func(t *testing.T) {
		inv := Invite{
			Code:      shortuuid.New(),
			Created:   time.Time{},
			ValidTill: time.Date(2025, 1, 2, 8, 37, 1, 1, time.UTC),
		}
		// So we can easily check is the expiry time is included (which is 0001-01-01).
		for strings.Contains(inv.Code, "1") {
			inv.Code = shortuuid.New()
		}

		msg, err := e.constructExpiry(inv, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, inv.Code) {
				t.Fatalf("code not found in output: %s", content)
			}
			if !strings.Contains(content, "02/01/25") || !strings.Contains(content, "08:37") {
				t.Fatalf("expiry not found in output: %s", content)
			}
		}
	})
}

// constructCreated(code, username, address string, invite Invite, placeholders bool)
func TestCreated(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	// Fix date/time format
	datePattern = "%d/%m/%y"
	timePattern = "%H:%M"
	testContent(e, customContent["UserCreated"], t, func(t *testing.T) {
		inv := Invite{
			Code:      shortuuid.New(),
			Created:   time.Time{},
			ValidTill: time.Date(2025, 1, 2, 8, 37, 1, 1, time.UTC),
		}
		username := shortuuid.New()
		address := shortuuid.New()

		msg, err := e.constructCreated(username, address, inv.ValidTill, inv, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, inv.Code) {
				t.Fatalf("code not found in output: %s", content)
			}
			if !strings.Contains(content, username) {
				t.Fatalf("username not found in output: %s", content)
			}
			if !strings.Contains(content, address) {
				t.Fatalf("address not found in output: %s", content)
			}
			if !strings.Contains(content, "02/01/25") || !strings.Contains(content, "08:37") {
				t.Fatalf("expiry not found in output: %s", content)
			}
		}
	})
}

// constructReset(pwr PasswordReset, placeholders bool)
func TestReset(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	// Fix date/time format
	datePattern = "%d/%m/%y"
	timePattern = "%H:%M"
	testContent(e, customContent["PasswordReset"], t, func(t *testing.T) {
		pwr := PasswordReset{
			Pin:      shortuuid.New(),
			Username: shortuuid.New(),
			Expiry:   time.Date(2025, 1, 2, 8, 37, 1, 1, time.UTC),
			Internal: false,
		}

		msg, err := e.constructReset(pwr, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, pwr.Pin) {
				t.Fatalf("pin not found in output: %s", content)
			}
			if !strings.Contains(content, pwr.Username) {
				t.Fatalf("username not found in output: %s", content)
			}
			if !strings.Contains(content, "02/01/25") || !strings.Contains(content, "08:37") {
				t.Fatalf("expiry not found in output: %s", content)
			}
		}
	})
}

// constructDeleted(reason string, placeholders bool)
func TestDeleted(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	testContent(e, customContent["UserDeleted"], t, func(t *testing.T) {
		reason := shortuuid.New()
		username := shortuuid.New()
		msg, err := e.constructDeleted(username, reason, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, reason) {
				t.Fatalf("reason not found in output: %s", content)
			}
			if !strings.Contains(content, username) {
				t.Fatalf("username not found in output: %s", content)
			}
		}
	})
}

// constructDisabled(reason string, placeholders bool)
func TestDisabled(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	testContent(e, customContent["UserDeleted"], t, func(t *testing.T) {
		reason := shortuuid.New()
		username := shortuuid.New()
		msg, err := e.constructDisabled(username, reason, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, reason) {
				t.Fatalf("reason not found in output: %s", content)
			}
			if !strings.Contains(content, username) {
				t.Fatalf("username not found in output: %s", content)
			}
		}
	})
}

// constructEnabled(reason string, placeholders bool)
func TestEnabled(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	testContent(e, customContent["UserDeleted"], t, func(t *testing.T) {
		reason := shortuuid.New()
		username := shortuuid.New()
		msg, err := e.constructEnabled(username, reason, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, reason) {
				t.Fatalf("reason not found in output: %s", content)
			}
			if !strings.Contains(content, username) {
				t.Fatalf("username not found in output: %s", content)
			}
		}
	})
}

// constructExpiryAdjusted(username string, expiry time.Time, reason string, placeholders bool)
func TestExpiryAdjusted(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	// Fix date/time format
	datePattern = "%d/%m/%y"
	timePattern = "%H:%M"
	testContent(e, customContent["UserExpiryAdjusted"], t, func(t *testing.T) {
		username := shortuuid.New()
		expiry := time.Date(2025, 1, 2, 8, 37, 1, 1, time.UTC)
		reason := shortuuid.New()
		msg, err := e.constructExpiryAdjusted(username, expiry, reason, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, username) {
				t.Fatalf("username not found in output: %s", content)
			}
			if !strings.Contains(content, reason) {
				t.Fatalf("reason not found in output: %s", content)
			}
			if !strings.Contains(content, "02/01/25") || !strings.Contains(content, "08:37") {
				t.Fatalf("expiry not found in output: %s", content)
			}
		}
	})
}

// constructExpiryReminder(username string, expiry time.Time, placeholders bool)
func TestExpiryReminder(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	// Fix date/time format
	datePattern = "%d/%m/%y"
	timePattern = "%H:%M"
	testContent(e, customContent["ExpiryReminder"], t, func(t *testing.T) {
		username := shortuuid.New()
		expiry := time.Date(2025, 1, 2, 8, 37, 1, 1, time.UTC)
		msg, err := e.constructExpiryReminder(username, expiry, false)
		if err != nil {
			t.Fatalf("failed construct: %+v", err)
		}
		for _, content := range []string{msg.Text, msg.HTML} {
			if !strings.Contains(content, username) {
				t.Fatalf("username not found in output: %s", content)
			}
			if !strings.Contains(content, "02/01/25") || !strings.Contains(content, "08:37") {
				t.Fatalf("expiry not found in output: %s", content)
			}
		}
	})
}

// constructWelcome(username string, expiry time.Time, placeholders bool)
func TestWelcome(t *testing.T) {
	e := testDummyEmailerInit(t)
	defer dbClose(e)
	if db == nil {
		t.Fatalf("db nil")
	}
	// Fix date/time format
	datePattern = "%d/%m/%y"
	timePattern = "%H:%M"
	testContent(e, customContent["WelcomeEmail"], t, func(t *testing.T) {
		username := shortuuid.New()
		expiry := time.Date(2025, 1, 2, 8, 37, 1, 1, time.UTC)
		msg, err := e.constructWelcome(username, expiry, false)
		t.Run("NoExpiry", func(t *testing.T) {
			if err != nil {
				t.Fatalf("failed construct: %+v", err)
			}
			for _, content := range []string{msg.Text, msg.HTML} {
				if !strings.Contains(content, username) {
					t.Fatalf("username not found in output: %s", content)
				}
				// time.Time{} is 0001-01-01... so look for a 1 in there at least.
				if !strings.Contains(content, "02/01/25") || !strings.Contains(content, "08:37") {
					t.Fatalf("expiry not found in output: %s", content)
				}
			}
		})
		username = shortuuid.New()
		expiry = time.Time{}
		msg, err = e.constructWelcome(username, expiry, false)
		t.Run("WithExpiry", func(t *testing.T) {

			if err != nil {
				t.Fatalf("failed construct: %+v", err)
			}
			for _, content := range []string{msg.Text, msg.HTML} {
				if !strings.Contains(content, username) {
					t.Fatalf("username not found in output: %s", content)
				}
				if strings.Contains(content, "01/01/01") || strings.Contains(content, "00:00") {
					t.Fatalf("empty expiry found in output: %s", content)
				}
			}
		})
	})
}
