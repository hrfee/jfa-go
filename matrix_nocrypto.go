//go:build !e2ee
// +build !e2ee

package main

import (
	"github.com/hrfee/jfa-go/logger"
	"maunium.net/go/mautrix/id"
)

type Crypto struct{}

func BuildTagsE2EE() {}

func MatrixE2EE() bool { return false }

func InitMatrixCrypto(d *MatrixDaemon, logger *logger.Logger) (err error) {
	d.Encryption = false
	return
}

func EncryptRoom(d *MatrixDaemon, roomID id.RoomID) error { return nil }
