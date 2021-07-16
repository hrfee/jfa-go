// +build !e2ee

package main

import (
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Crypto struct{}

func MatrixE2EE() bool { return false }

func InitMatrixCrypto(d *MatrixDaemon) (err error) {
	d.Encryption = false
	return
}

func HandleSyncerCrypto(startTime int64, d *MatrixDaemon, syncer *mautrix.DefaultSyncer) {
	return
}

func CryptoShutdown(d *MatrixDaemon) {
	return
}

func EncryptRoom(d *MatrixDaemon, room *mautrix.RespCreateRoom, userID id.UserID) (encrypted bool) {
	return
}

func SendEncrypted(d *MatrixDaemon, content *event.MessageEventContent, roomID id.RoomID) (err error) {
	err = d.send(content, roomID)
	return
}
