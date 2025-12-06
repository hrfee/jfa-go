//go:build e2ee
// +build e2ee

package main

import (
	"context"

	"github.com/hrfee/jfa-go/logger"
	lm "github.com/hrfee/jfa-go/logmessages"
	_ "github.com/mattn/go-sqlite3"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Crypto struct {
	helper *cryptohelper.CryptoHelper
}

func BuildTagsE2EE() {
	buildTags = append(buildTags, "e2ee")
}

func MatrixE2EE() bool { return true }

func InitMatrixCrypto(d *MatrixDaemon, logger *logger.Logger) error {
	logger.Printf(lm.InitingMatrixCrypto)
	d.Encryption = d.app.config.Section("matrix").Key("encryption").MustBool(false)
	if !d.Encryption {
		// return fmt.Errorf("encryption disabled")
		return nil
	}

	dbPath := d.app.config.Section("files").Key("matrix_sql").String()
	var err error
	d.crypto = &Crypto{}
	d.crypto.helper, err = cryptohelper.NewCryptoHelper(d.bot, []byte("jfa-go"), dbPath)
	if err != nil {
		return err
	}

	err = d.crypto.helper.Init(context.TODO())
	if err != nil {
		return err
	}

	d.bot.Crypto = d.crypto.helper

	d.Encryption = true
	logger.Printf(lm.InitMatrixCrypto)
	return nil
}

func EncryptRoom(d *MatrixDaemon, roomID id.RoomID) error {
	if !d.Encryption {
		return nil
	}
	_, err := d.bot.SendStateEvent(context.TODO(), roomID, event.StateEncryption, "", event.EncryptionEventContent{
		Algorithm:              id.AlgorithmMegolmV1,
		RotationPeriodMillis:   7 * 24 * 60 * 60 * 1000,
		RotationPeriodMessages: 100,
	})
	return err
}
