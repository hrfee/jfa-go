package main

import (
	"fmt"
	"strings"

	"maunium.net/go/mautrix"
	gomatrix "maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type stateStore struct {
	isEncrypted *map[id.RoomID]bool
}

func (m *stateStore) IsEncrypted(roomID id.RoomID) bool {
	// encrypted, ok := (*m.isEncrypted)[roomID]
	// fmt.Println("ISENCRYPTED", roomID, ok && encrypted)
	// return ok && encrypted
	return true
}

func (m *stateStore) GetEncryptionEvent(roomID id.RoomID) *event.EncryptionEventContent {
	return &event.EncryptionEventContent{
		Algorithm:              id.AlgorithmMegolmV1,
		RotationPeriodMillis:   7 * 24 * 60 * 60 * 1000,
		RotationPeriodMessages: 100,
	}
}

// Users are assumed to only have one common channel with the bot, so we can stub this out.
func (m *stateStore) FindSharedRooms(userID id.UserID) []id.RoomID {
	// for _, user := range m.app.storage.matrix {
	// 	if id.UserID(user.UserID) == userID {
	// 		return []id.RoomID{id.RoomID(user.RoomID)}
	// 	}
	// }
	return []id.RoomID{}
}

func (d *MatrixDaemon) getUserIDs(roomID id.RoomID) (list []id.UserID, err error) {
	members, err := d.bot.JoinedMembers(roomID)
	if err != nil {
		return
	}
	list = make([]id.UserID, len(members.Joined))
	i := 0
	for id := range members.Joined {
		list[i] = id
		i++
	}
	return
}

type olmLogger struct {
	app *appContext
}

func (o olmLogger) Error(message string, args ...interface{}) {
	o.app.err.Printf("OLM: "+message+"\n", args)
}

func (o olmLogger) Warn(message string, args ...interface{}) {
	o.app.info.Printf("OLM: "+message+"\n", args)
}

func (o olmLogger) Debug(message string, args ...interface{}) {
	o.app.debug.Printf("OLM: "+message+"\n", args)
}

func (o olmLogger) Trace(message string, args ...interface{}) {
	if strings.HasPrefix(message, "Got membership state event") {
		return
	}
	o.app.debug.Printf("OLM [TRACE]: "+message+"\n", args)
}

func InitMatrixCrypto(d *MatrixDaemon) (err error) {
	d.Encryption = d.app.config.Section("matrix").Key("encryption").MustBool(false)
	if !d.Encryption {
		return
	}
	for _, user := range d.app.storage.matrix {
		d.isEncrypted[id.RoomID(user.RoomID)] = user.Encrypted
	}
	dbPath := d.app.config.Section("files").Key("matrix_sql").String()
	// If the db is maintained after restart, element reports "The secure channel with the sender was corrupted" when sending a message from the bot.
	// This obviously isn't right, but it seems to work.
	// Since its not really used anyway, just use the deprecated GobStore. This reduces cgo usage anyway.
	// os.Remove(dbPath)
	var cryptoStore *crypto.GobStore
	cryptoStore, err = crypto.NewGobStore(dbPath)
	// d.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return
	}
	olmLog := &olmLogger{d.app}
	// deviceID := "jfa-go" + commit
	// cryptoStore := crypto.NewSQLCryptoStore(d.db, "sqlite3", string(d.userID)+deviceID, id.DeviceID(deviceID), []byte("jfa-go"), olmLog)
	// err = cryptoStore.CreateTables()
	// if err != nil {
	// 	return
	// }
	d.olm = crypto.NewOlmMachine(d.bot, olmLog, cryptoStore, &stateStore{&d.isEncrypted})
	d.olm.AllowUnverifiedDevices = true
	err = d.olm.Load()
	if err != nil {
		return
	}
	d.cryptoStore = cryptoStore
	return
}

func HandleSyncerCrypto(startTime int64, d *MatrixDaemon, syncer *mautrix.DefaultSyncer) {
	if !d.Encryption {
		return
	}
	syncer.OnSync(func(resp *gomatrix.RespSync, since string) bool {
		d.olm.ProcessSyncResponse(resp, since)
		return true
	})
	syncer.OnEventType(event.StateMember, func(source gomatrix.EventSource, evt *event.Event) {
		d.olm.HandleMemberEvent(evt)
		// if evt.Content.AsMember().Membership != event.MembershipJoin {
		// 	return
		// }
		// userIDs, err := d.getUserIDs(evt.RoomID)
		// if err != nil || len(userIDs) < 2 {
		// 	fmt.Println("FS", err)
		// 	return
		// }
		// err = d.olm.ShareGroupSession(evt.RoomID, userIDs)
		// if err != nil {
		// 	fmt.Println("FS", err)
		// 	return
		// }
	})
	syncer.OnEventType(event.EventEncrypted, func(source gomatrix.EventSource, evt *event.Event) {
		if evt.Timestamp < startTime {
			return
		}
		fmt.Printf("%+v\n", d.cryptoStore.GroupSessions)
		decrypted, err := d.olm.DecryptMegolmEvent(evt)
		if err != nil {
			d.app.err.Printf("Failed to decrypt Matrix message: %v", err)
			return
		}
		fmt.Println("RECV", decrypted.Content.Raw["body"])
		d.handleMessage(source, decrypted)
	})
}

func CryptoShutdown(d *MatrixDaemon) {
	if d.Encryption {
		d.olm.FlushStore()
	}
}

func EncryptRoom(d *MatrixDaemon, room *mautrix.RespCreateRoom, userID id.UserID) (encrypted bool) {
	if !d.Encryption {
		return
	}
	_, err := d.bot.SendStateEvent(room.RoomID, event.StateEncryption, "", &event.EncryptionEventContent{
		Algorithm:              id.AlgorithmMegolmV1,
		RotationPeriodMillis:   7 * 24 * 60 * 60 * 1000,
		RotationPeriodMessages: 100,
	})
	if err == nil {
		encrypted = true
	} else {
		d.app.debug.Printf("Matrix: Failed to enable encryption in room: %v", err)
		return
	}
	d.isEncrypted[room.RoomID] = encrypted
	var userIDs []id.UserID
	userIDs, err = d.getUserIDs(room.RoomID)
	if err != nil {
		return
	}
	userIDs = append(userIDs, userID)
	err = d.olm.ShareGroupSession(room.RoomID, userIDs)
	return
}

func SendEncrypted(d *MatrixDaemon, content *event.MessageEventContent, roomID id.RoomID) (err error) {
	if !d.Encryption {
		err = d.send(content, roomID)
		return
	}
	var encrypted *event.EncryptedEventContent
	encrypted, err = d.olm.EncryptMegolmEvent(roomID, event.EventMessage, content)
	if err == crypto.SessionExpired || err == crypto.SessionNotShared || err == crypto.NoGroupSession {
		fmt.Println("SGS")
		// err = d.olm.ShareGroupSession(id.RoomID(user.RoomID), []id.UserID{id.UserID(user.UserID), d.userID})
		var userIDs []id.UserID
		userIDs, err = d.getUserIDs(roomID)
		if err != nil {
			return
		}
		fmt.Println("SHARETO", userIDs)
		err = d.olm.ShareGroupSession(roomID, userIDs)
		if err != nil {
			return
		}
		encrypted, err = d.olm.EncryptMegolmEvent(roomID, event.EventMessage, content)
	}
	if err != nil {
		return
	}
	_, err = d.bot.SendMessageEvent(roomID, event.EventEncrypted, &event.Content{Parsed: encrypted})
	if err != nil {
		return
	}
	return
}
