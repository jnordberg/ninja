package main

import (
	"bitbucket.org/ckvist/twilio/twirest"
	log "github.com/Sirupsen/logrus"
	"github.com/yvasiyarov/gorelic"
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

	if Env.Vars.NewrelicEnable {
		Env.NRAgent = gorelic.NewAgent()
		Env.NRAgent.NewrelicLicense = Env.Vars.NewrelicKey
		Env.NRAgent.Verbose = Env.Vars.NewrelicDebug
		Env.NRAgent.Run()
	}

	go SetupDatabase()
	SetupBot()
	SetupWeb()
}
