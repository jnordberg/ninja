package main

import (
	"bitbucket.org/ckvist/twilio/twiml"
	"bitbucket.org/ckvist/twilio/twirest"
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"math/rand"
	"net/http"
	"ninja/slack"
	"os"
	"strings"
	"time"
)

var bot *slack.Bot
var dbSession *mgo.Session
var dbName string
var tClient *twirest.TwilioClient
var fromNumber string

var codeChars = []rune("abcdefghjkmnpqrstuvwxyz")

func randCode(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = codeChars[rand.Intn(len(codeChars))]
	}
	return string(b)
}

type User struct {
	Id         bson.ObjectId `bson:"_id,omitempty"`
	UserId     string        `bson:"user_id"`
	Name       string        `bson:"name"`
	Phone      string        `bson:"phone"`
	PhoneValid bool          `bson:"phone_valid"`
	PhoneCode  string        `bson:"phone_code"`
	Runner     bool          `bson:"runner"`
}

func getUser(m *slack.IncomingMessage) (*User, error) {
	if dbSession == nil {
		return nil, errors.New("database not ready")
	}

	s := dbSession.Clone()
	c := s.DB(dbName).C("users")
	defer s.Close()

	user := User{}
	err := c.Find(bson.M{"user_id": m.UserId}).One(&user)
	if err == nil {
		return &user, nil
	} else {
		user.Id = bson.NewObjectId()
		user.UserId = m.UserId
		user.Name = m.UserName
		user.Runner = false
		user.PhoneValid = false

		err := c.Insert(&user)
		if err != nil {
			return nil, err
		} else {
			return &user, nil
		}
	}
}

func sendCode(user *User) error {
	log.Printf("Sending code '%s' to %s on %s", user.PhoneCode, user.Name, user.Phone)

	text := fmt.Sprintf(
		"Hey %s! Ninja here, you need to verify this number."+
			"To do that just write the following in the #coffee channel:\n\nverify %s",
		user.Name, user.PhoneCode,
	)

	msg := twirest.SendMessage{
		Text: text,
		To:   user.Phone,
		From: fromNumber,
	}

	resp, err := tClient.Request(msg)
	if err != nil {
		return err
	}

	log.Println("Response from twilio: ", resp.Message.Status)
	return nil
}

func validateCode(user *User, code string) (*slack.OutgoingMessage, error) {
	var err error

	if !user.Runner {
		return slack.NewMessage("What are you on about? You need to register first!"), nil
	}

	if user.PhoneValid {
		return slack.NewMessage("You have already verified your phone, relax!"), nil
	}

	if user.PhoneCode == strings.ToLower(code) {
		s := dbSession.Clone()
		c := s.DB(dbName).C("users")
		defer s.Close()

		user.PhoneValid = true
		err = c.UpdateId(user.Id, &user)
		if err != nil {
			return nil, err
		}

		call := twirest.MakeCall{
			Url:  os.Getenv("APP_URL") + "/call",
			To:   user.Phone,
			From: fromNumber,
		}

		_, err = tClient.Request(call)
		if err != nil {
			return nil, err
		}

		var msg string = fmt.Sprintf("Hehehe %s, I've got your number now! :)", user.Name)
		return slack.NewMessage(msg), nil
	} else {
		return slack.NewMessage("Hmm... That's not the right code you know."), nil
	}

}

func updatePhone(user *User, phone string) (*slack.OutgoingMessage, error) {
	var err error

	if !strings.HasPrefix(phone, "+") {
		return slack.NewMessage("Invalid phone number, you must specify the country-code. e.g. `+61488888888`"), nil
	}

	if user.Phone == phone {
		if user.PhoneValid {
			return slack.NewMessage("I've got your phone number already."), nil
		} else {
			err = sendCode(user)
			return slack.NewMessage("I've got that number, but you need to validate it. I'll resend the code..."), err
		}
	}

	s := dbSession.Clone()
	c := s.DB(dbName).C("users")
	defer s.Close()

	var msg string
	if user.Runner {
		msg = fmt.Sprintf("Ok %s, got your new number. You need to validate it, I'll send you a text.", user.Name)
	} else {
		msg = fmt.Sprintf("Thanks %s! You're now a coffee-runner. Check your phone for instructions.", user.Name)
	}

	user.Phone = phone
	user.PhoneValid = false
	user.PhoneCode = randCode(6)
	user.Runner = true

	err = c.UpdateId(user.Id, &user)
	if err != nil {
		return nil, err
	}

	err = sendCode(user)

	return slack.NewMessage(msg), err
}

func handleMessage(m *slack.IncomingMessage) (*slack.OutgoingMessage, error) {
	log.Println("message", m.Text)

	parts := strings.Split(strings.TrimSpace(m.Text), " ")

	if len(parts) < 2 {
		return nil, nil
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	// TODO: only get user for valid commands?
	user, err := getUser(m)
	if err != nil {
		return nil, err
	}

	switch command {
	case "register":
		return updatePhone(user, args[0])
	case "verify":
		return validateCode(user, args[0])
	default:
		return nil, nil
	}
}

func setupMongo(url string) {
	log.Println("Connecting to mongodb")
	s, err := mgo.Dial(url)
	if err != nil {
		panic(err)
	}
	log.Printf("Connected to database (%s)", dbName)
	dbSession = s
}

func callHandler(w http.ResponseWriter, r *http.Request) {
	resp := twiml.NewResponse()
	resp.Action(twiml.Play{
		Url: os.Getenv("APP_URL") + "/assets/roll.mp3",
	})
	resp.Send(w)
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "CoffeeNinja at your service")
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mongo_url := os.Getenv("MONGOHQ_URL")
	if mongo_url == "" {
		mongo_url = "mongodb://localhost/ninja"
	}

	urlParts := strings.Split(mongo_url, "/")
	dbName = urlParts[len(urlParts)-1]

	go setupMongo(mongo_url)

	bot = &slack.Bot{
		Subdomain:      os.Getenv("SLACK_DOMAIN"),
		Token:          os.Getenv("SLACK_TOKEN"),
		MessageHandler: handleMessage,
	}

	tClient = twirest.NewClient(os.Getenv("TWILIO_SID"), os.Getenv("TWILIO_TOKEN"))
	fromNumber = os.Getenv("TWILIO_NUMBER")

	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/slack", bot.SlackHandler)
	http.HandleFunc("/call", callHandler)
	http.HandleFunc("/assets/", staticHandler)

	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
