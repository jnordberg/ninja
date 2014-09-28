package main

import (
	"bitbucket.org/ckvist/twilio/twirest"
	log "github.com/Sirupsen/logrus"
	"math/rand"
	"time"
)

func main() {
	LoadEnv()

	rand.Seed(time.Now().UTC().UnixNano())

	log.SetFormatter(&log.TextFormatter{ForceColors: Env.Vars.ForceColors})
	level, err := log.ParseLevel(Env.Vars.LogLevel)
	if err != nil {
		log.Panic(err)
	}
	log.SetLevel(level)

	Env.TwiClient = twirest.NewClient(Env.Vars.TwilioSID, Env.Vars.TwilioToken)

	go SetupDatabase()
	SetupBot()
	SetupWeb()
}
