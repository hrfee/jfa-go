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
	// bmss, err := NewBackedMemoryStateStore(d.app.storage.db)
	// if err != nil {
	// 	return err
	// }
	// d.bot.StateStore = bmss
	d.crypto.helper, err = cryptohelper.NewCryptoHelper(d.bot, []byte("jfa-go"), dbPath)
	// bms, err := NewBackedMemoryStore(d.app.storage.db)
	// if err != nil {
	// 	return err
	// }
	// d.crypto.helper, err = cryptohelper.NewCryptoHelper(d.bot, []byte("jfa-go"), bms)
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

/*type BackedMemoryStore struct {
	*crypto.MemoryStore
	db *badgerhold.Store
}

func (b *BackedMemoryStore) save() error {
	err := b.db.Upsert("MatrixEncryptionStore", b.MemoryStore)
	defer func(err error) { log.Printf("MATRIX WRITE: err=%v\n", err) }(err)
	return err
}

func NewBackedMemoryStore(db *badgerhold.Store) (*BackedMemoryStore, error) {
	b := &BackedMemoryStore{
		db: db,
	}
	b.MemoryStore = crypto.NewMemoryStore(b.save)
	err := b.db.Get("MatrixEncryptionStore", b.MemoryStore)
	if err != nil && !errors.Is(err, badgerhold.ErrNotFound) {
		return nil, err
	}
	return b, nil
}

type BackedMemoryStateStore struct {
	*mautrix.MemoryStateStore
	db *badgerhold.Store
}

func (b *BackedMemoryStateStore) save() error {
	err := b.db.Upsert("MatrixEncryptionStateStore", b.MemoryStateStore)
	defer func(err error) { log.Printf("MATRIX WRITE: err=%v\n", err) }(err)
	return err
}

func NewBackedMemoryStateStore(db *badgerhold.Store) (*BackedMemoryStateStore, error) {
	b := &BackedMemoryStateStore{
		db: db,
	}

	store := mautrix.NewMemoryStateStore()
	memStore, ok := store.(*mautrix.MemoryStateStore)
	if !ok {
		return nil, errors.New("didn't get a MemoryStateStore")
	}
	b.MemoryStateStore = memStore
	err := b.db.Get("MatrixEncryptionStateStore", b.MemoryStateStore)
	if err != nil && !errors.Is(err, badgerhold.ErrNotFound) {
		return nil, err
	}
	return b, nil
}*/
