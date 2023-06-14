package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hrfee/mediabrowser"
)

var (
	names = []string{"Aaron", "Agnes", "Bridget", "Brandon", "Dolly", "Drake", "Elizabeth", "Erika", "Geoff", "Graham", "Haley", "Halsey", "Josie", "John", "Kayleigh", "Luka", "Melissa", "Nasreen", "Paul", "Ross", "Sam", "Talib", "Veronika", "Zaynab"}
)

const (
	PASSWORD = "test"
	COUNT    = 10
)

func main() {
	fmt.Println("Usage: account-gen <server> <username> <password, or file://path to file containing password>")
	var server, username, password string
	reader := bufio.NewReader(os.Stdin)
	if len(os.Args) > 1 {
		server = os.Args[1]
	} else {
		fmt.Print("Server Address: ")
		server, _ = reader.ReadString('\n')
		server = strings.TrimSuffix(server, "\n")
	}

	if len(os.Args) > 2 {
		username = os.Args[2]
	} else {
		fmt.Print("Username: ")
		username, _ = reader.ReadString('\n')
		username = strings.TrimSuffix(username, "\n")
	}

	if len(os.Args) > 3 {
		password = os.Args[3]
		if strings.HasPrefix(password, "file://") {
			p, err := os.ReadFile(strings.TrimPrefix(password, "file://"))
			if err != nil {
				log.Fatalf("Failed to read password file \"%s\": %+v\n", password, err)
			}
			password = strings.TrimSuffix(string(p), "\n")
		}
	} else {
		fmt.Print("Password: ")
		password, _ = reader.ReadString('\n')
		password = strings.TrimSuffix(password, "\n")
	}

	jf, err := mediabrowser.NewServer(
		mediabrowser.JellyfinServer,
		server,
		"jfa-go-account-gen-script",
		"0.0.1",
		"testing",
		"my_left_foot",
		mediabrowser.NewNamedTimeoutHandler("Jellyfin Account Gen", "\""+server+"\"", true),
		30,
	)

	if err != nil {
		log.Fatalf("Failed to connect to Jellyin @ \"%s\": %+v\n", server, err)
	}

	_, status, err := jf.Authenticate(username, password)
	if status != 200 || err != nil {
		log.Fatalf("Failed to authenticate: %+v\n", err)
	}

	jfTemp, err := mediabrowser.NewServer(
		mediabrowser.JellyfinServer,
		server,
		"jfa-go-account-gen-script",
		"0.0.1",
		"fake-activity",
		"my_left_foot",
		mediabrowser.NewNamedTimeoutHandler("Jellyfin Account Gen", "\""+server+"\"", true),
		30,
	)

	if err != nil {
		log.Fatalf("Failed to connect to Jellyin @ \"%s\": %+v\n", server, err)
	}

	rand.Seed(time.Now().Unix())

	for i := 0; i < COUNT; i++ {
		name := names[rand.Intn(len(names))] + strconv.Itoa(rand.Intn(100))

		user, status, err := jf.NewUser(name, PASSWORD)
		if (status != 200 && status != 201 && status != 204) || err != nil {
			log.Fatalf("Failed to create user \"%s\" (%d): %+v\n", name, status, err)
		}

		if rand.Intn(100) > 65 {
			user.Policy.IsAdministrator = true
		}

		if rand.Intn(100) > 80 {
			user.Policy.IsDisabled = true
		}

		status, err = jf.SetPolicy(user.ID, user.Policy)
		if (status != 200 && status != 201 && status != 204) || err != nil {
			log.Fatalf("Failed to set policy for user \"%s\" (%d): %+v\n", name, status, err)
		}

		if rand.Intn(100) > 20 {
			jfTemp.Authenticate(name, PASSWORD)
		}
	}
}
