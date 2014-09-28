package main

import (
	"bitbucket.org/ckvist/twilio/twirest"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"ninja/slack"
	"regexp"
	"strings"
	"time"
)

type ArgMap map[string]string
type CmdFunc func(args ArgMap, user *User, m *slack.IncomingMessage) *slack.OutgoingMessage
type Command struct {
	Pattern *regexp.Regexp
	Handler CmdFunc
}

var Commands []Command

func SendCode(user *User) error {
	log.Infof("Sending code '%s' to %s on %s", user.PhoneCode, user.Name, user.Phone)

	text := fmt.Sprintf(
		"Hey %s! Ninja here, you need to verify this number."+
			"To do that just write the following in the #coffee channel:\n\nverify %s",
		user.Name, user.PhoneCode,
	)

	msg := twirest.SendMessage{
		Text: text,
		To:   user.Phone,
		From: Env.Vars.TwilioNumber,
	}

	resp, err := Env.TwiClient.Request(msg)
	if err != nil {
		return err
	}

	log.Debugf("Response from twilio: ", resp.Message.Status)
	return nil
}

func HelpCommand(args ArgMap, user *User, m *slack.IncomingMessage) *slack.OutgoingMessage {
	log.Println(args)
	return slack.NewMessage("```" +
		"Ninja words\n" +
		"---------------------------------------------------\n" +
		"help                             you'll never guess\n" +
		"register <phone#>                    become a ninja\n" +
		"verify <code>                    verify your karate\n" +
		"startrun                         start a coffee-run\n" +
		"done                                     finish run\n" +
		"```")
}

func RegisterCommand(args ArgMap, user *User, m *slack.IncomingMessage) *slack.OutgoingMessage {
	phone := strings.Replace(args["phone"], " ", "", -1)

	if !strings.HasPrefix(phone, "+") {
		return slack.NewMessage("Invalid phone number, you must specify the country-code. e.g. `+61488888888`")
	}

	if user.Phone == phone {
		if user.PhoneValid {
			return slack.NewMessage("I've got your phone number already.")
		} else {
			if err := SendCode(user); err != nil {
				return slack.ErrorMessage(err)
			}
			return slack.NewMessage("I've got that number, but you need to validate it. I'll resend the code...")
		}
	}

	c := GetCollection("users")
	was_runner := user.Runner

	user.Phone = phone
	user.PhoneValid = false
	user.PhoneCode = GenerateCode(6)
	user.Runner = true

	if err := c.UpdateId(user.Id, &user); err != nil {
		return slack.ErrorMessage(err)
	}

	if err := SendCode(user); err != nil {
		return slack.ErrorMessage(err)
	}

	var msg string
	if was_runner {
		msg = fmt.Sprintf("Ok %s, got your new number. You need to validate it, I'll send you a text.", user.Name)
	} else {
		msg = fmt.Sprintf("Thanks %s! You're now a coffee-runner. Check your phone for instructions.", user.Name)
	}

	return slack.NewMessage(msg)
}

func VerifyCommand(args ArgMap, user *User, m *slack.IncomingMessage) *slack.OutgoingMessage {
	if !user.Runner {
		return slack.NewMessage("What are you on about? You need to register first!")
	}

	if user.PhoneValid {
		return slack.NewMessage("You have already verified your phone, relax!")
	}

	code := strings.ToLower(strings.TrimSpace(args["code"]))

	if user.PhoneCode == code {
		log.Infof("User %s verified their phone", user.Name)

		c := GetCollection("users")

		user.PhoneValid = true

		if err := c.UpdateId(user.Id, &user); err != nil {
			return slack.ErrorMessage(err)
		}

		call := twirest.MakeCall{
			Url:  Env.Vars.AppURL + "/call",
			To:   user.Phone,
			From: Env.Vars.TwilioNumber,
		}

		if _, err := Env.TwiClient.Request(call); err != nil {
			return slack.ErrorMessage(err)
		}

		return slack.NewMessage(fmt.Sprintf("Hehehe %s... I've got your number now! :)", user.Name))
	} else {
		return slack.NewMessage("Hmm... That's not the right code you know.")
	}
}

func StartCommand(args ArgMap, user *User, m *slack.IncomingMessage) *slack.OutgoingMessage {
	if !user.Runner || !user.PhoneValid {
		return slack.NewMessage("You're not a runner, register first.")
	}

	if Env.ActiveRun != nil {
		return slack.NewMessage("Already in an active run, please wait for it to finish.")
	}

	run := &Run{}
	run.Id = bson.NewObjectId()
	run.Runner = user.Id
	run.Items = []Item{}

	log.Printf("run %#v", run)

	c := GetCollection("runs")

	if err := c.Insert(run); err != nil {
		return slack.ErrorMessage(err)
	}

	Env.ActiveRun = &run.Id

	msg := fmt.Sprintf(
		"<!channel> %s is starting a coffee-run! Type `order <coffee type>` to get yours. "+
			"You have 10 minutes or until %s writes `done`.",
		user.Name, user.Name,
	)

	go RunTimer()
	go RemindTimer()

	return slack.NewMessage(msg)
}

func EndRun(user *User) *slack.OutgoingMessage {
	if Env.ActiveRun == nil {
		return nil
	}

	run := Run{}
	c := GetCollection("runs")

	if err := c.FindId(Env.ActiveRun).One(&run); err != nil {
		return slack.ErrorMessage(err)
	}

	if user == nil {
		q := GetCollection("users").Find(bson.M{"user_id": run.Runner})
		if err := q.One(&user); err != nil {
			log.Panic(err)
		}
	}

	Env.ActiveRun = nil

	if len(run.Items) == 0 {
		return slack.NewMessage("No one ordered :crying_cat_face:")
	}

	msg := fmt.Sprintf("Ordering done! %s will now fetch your coffees. 1+ coffee karma.\n```", user.Name)
	sms := "Coffee!\n"
	for i := 0; i < len(run.Items); i++ {
		item := run.Items[i]
		msg += fmt.Sprintf("\n%s: %s", item.OwnerName, item.Name)
		sms += fmt.Sprintf("\n%s: %s", item.OwnerName, item.Name)
	}

	msg += "```"

	req := twirest.SendMessage{
		Text: sms,
		To:   user.Phone,
		From: Env.Vars.TwilioNumber,
	}

	resp, err := Env.TwiClient.Request(req)
	if err != nil {
		return slack.ErrorMessage(err)
	}

	log.Debugf("Response from twilio: ", resp.Message.Status)

	return slack.NewMessage(msg)
}

func RemindTimer() {
	var this_run string = Env.ActiveRun.String()
	time.AfterFunc(time.Minute*8, func() {
		if Env.ActiveRun.String() == this_run {
			Env.Bot.SendMessage(slack.NewMessage("<!channel> 2 minutes remaning, get your orders in!"))
		}
	})
}

func RunTimer() {
	time.AfterFunc(time.Minute*10, func() {
		if Env.ActiveRun == nil {
			log.Info("Run already ended.. nothing to do")
		} else {
			Env.Bot.SendMessage(EndRun(nil))
		}
	})
}

func DoneCommand(args ArgMap, user *User, m *slack.IncomingMessage) *slack.OutgoingMessage {
	if Env.ActiveRun == nil {
		return nil
	}
	return EndRun(user)
}

func OrderCommand(args ArgMap, user *User, m *slack.IncomingMessage) *slack.OutgoingMessage {
	if Env.ActiveRun == nil {
		return slack.NewMessage("No one is running, why not start a run yourself with `startrun`")
	}

	c := GetCollection("runs")

	item := Item{
		Name:      strings.TrimSpace(args["item"]),
		OwnerId:   user.Id,
		OwnerName: user.Name,
	}

	update := bson.M{"$push": bson.M{"items": item}}

	if err := c.UpdateId(Env.ActiveRun, update); err != nil {
		return slack.ErrorMessage(err)
	}

	return slack.NewMessage(fmt.Sprintf("%s wants a %s", user.Name, item.Name))
}

func AddCommand(pattern string, handler CmdFunc) {
	var cmd Command
	expr, err := regexp.Compile(pattern)
	if err != nil {
		log.Panic(err)
	}
	cmd.Pattern = expr
	cmd.Handler = handler
	Commands = append(Commands, cmd)
}

func GetUser(m *slack.IncomingMessage) *User {
	user := User{}
	c := GetCollection("users")
	q := c.Find(bson.M{"user_id": m.UserId})

	if err := q.One(&user); err != nil {
		if err == mgo.ErrNotFound {
			log.Infof("Creating new user %s (%s)", m.UserName, m.UserId)
			user.Id = bson.NewObjectId()
			user.UserId = m.UserId
			user.Name = m.UserName
			user.Runner = false
			user.PhoneValid = false
			if err := c.Insert(&user); err != nil {
				log.Panic(err)
			}
		} else {
			log.Panic(err)
		}
	}

	return &user
}

func BotHandler(m *slack.IncomingMessage) *slack.OutgoingMessage {
	text := strings.TrimSpace(m.Text)
	var cmd Command
	for i := 0; i < len(Commands); i++ {
		cmd = Commands[i]
		if cmd.Pattern.MatchString(text) {
			names := cmd.Pattern.SubexpNames()
			values := cmd.Pattern.FindStringSubmatch(text)
			args := make(ArgMap)
			for j := 1; j < len(values); j++ {
				args[names[j]] = values[j]
			}
			log.Debugf("matched command %s", cmd.Pattern)
			return cmd.Handler(args, GetUser(m), m)
		}

	}
	return nil
}

func SetupBot() {
	AddCommand("^help$", HelpCommand)
	AddCommand("^register (?P<phone>[+0-9 ]+)$", RegisterCommand)
	AddCommand("^verify (?P<code>.*)$", VerifyCommand)
	AddCommand("^startrun$", StartCommand)
	AddCommand("^order (?P<item>[a-zA-Z0-9 ]+)$", OrderCommand)
	AddCommand("^done$", DoneCommand)

	Env.Bot = &slack.Bot{
		Subdomain:      Env.Vars.SlackDomain,
		Token:          Env.Vars.SlackToken,
		MessageHandler: BotHandler,
	}
}
