package main

import (
	"fmt"
	"log"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	TOKEN = "1785754648:AAG4G6PKZpGDEJM_-MeQHJqD-xUDrrLrTC4"
	USER  = "johnikwock"

	AUTH = "AB-CD-EF"
)

func main() {
	log.Println("Connecting...")
	bot, err := tg.NewBotAPI(TOKEN)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}
	bot.Debug = false
	log.Printf("Authorized Telegram bot \"%s\"", bot.Self.UserName)

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		log.Printf("New message from \"@%s\": \"%s\"", update.Message.From.UserName, update.Message.Text)
		if update.Message.From.UserName != USER {
			continue
		}
		var msg tg.MessageConfig
		sects := strings.Split(update.Message.Text, " ")
		if sects[0] == "/start" {
			msg = tg.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Enter this code on the sign-up page to continue: %s", AUTH))
		} else if sects[0] != "/auth" || sects[len(sects)-1] != AUTH {
			log.Println("Invalid command or auth token")
			msg = tg.NewMessage(update.Message.Chat.ID, "Invalid command or token")
		} else {
			msg = tg.NewMessage(update.Message.Chat.ID, "Success!")
			log.Println("Successful auth")
		}
		msg.ReplyToMessageID = update.Message.MessageID

		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Send failed: %v", err)
		}
	}

}
