package main

import (
	"bitbucket.org/ckvist/twilio/twirest"
	"github.com/yvasiyarov/gorelic"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"ninja/slack"
	"os"
	"reflect"
	"strconv"
	"time"
)

type EnvVars struct {
	AppURL        string `env:"APP_URL"`
	ForceColors   bool   `env:"FORCE_COLORS" default:"false"`
	LogLevel      string `env:"LOG_LEVEL" default:"info"`
	MongoDB       string `env:"MONGO_DB"`
	MongoURL      string `env:"MONGOHQ_URL"`
	ServerPort    string `env:"PORT" default:"3000"`
	SlackDomain   string `env:"SLACK_DOMAIN"`
	SlackToken    string `env:"SLACK_TOKEN"`
	TwilioNumber  string `env:"TWILIO_NUMBER"`
	TwilioSID     string `env:"TWILIO_SID"`
	TwilioToken   string `env:"TWILIO_TOKEN"`
	NewrelicKey   string `env:"NEW_RELIC_LICENSE_KEY"`
	NewrelicDebug bool   `env:"NEW_RELIC_DEBUG"`
}

var Env struct {
	Bot       *slack.Bot
	DBSession *mgo.Session
	TwiClient *twirest.TwilioClient
	Vars      *EnvVars
	Started   time.Time
	ActiveRun *bson.ObjectId
	NRAgent   *gorelic.Agent
}

func LoadEnv() {
	Env.Vars = &EnvVars{}
	Env.Started = time.Now()
	v := reflect.ValueOf(Env.Vars).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)
		env_key := tf.Tag.Get("env")
		env_val := os.Getenv(env_key)
		if env_val == "" {
			env_val = tf.Tag.Get("default")
		}
		field := v.Field(i)
		switch field.Kind() {
		case reflect.String:
			field.SetString(env_val)
		case reflect.Int:
			val, _ := strconv.Atoi(env_val)
			field.SetInt(int64(val))
		case reflect.Bool:
			val, _ := strconv.ParseBool(env_val)
			field.SetBool(val)
		}
	}
}
