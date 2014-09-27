package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ajg/form"
	"log"
	"net/http"
)

const UrlTemplate string = "https://%s.slack.com/services/hooks/incoming-webhook?token=%s"

type IncomingMessage struct {
	ChannelId   string  `form:"channel_id"`
	ChannelName string  `form:"channel_name"`
	ServiceId   string  `form:"service_id"`
	TeamDomain  string  `form:"team_domain"`
	TeamId      string  `form:"team_id"`
	Text        string  `form:"text"`
	Timestamp   float32 `form:"timestamp"`
	Token       string  `form:"token"`
	TriggerWord string  `form:"trigger_word"`
	UserId      string  `form:"user_id"`
	UserName    string  `form:"user_name"`
}

type OutgoingMessage struct {
	Channel     string `json:"channel,omitempty"`
	From        string `json:"username,omitempty"`
	Text        string `json:"text"`
	UseMarkdown bool   `json:"mrkdwn,omitempty"`
}

type Bot struct {
	Subdomain      string
	Token          string
	MessageHandler func(m *IncomingMessage) (*OutgoingMessage, error)
	Debug          bool
}

func NewMessage(msg string) *OutgoingMessage {
	return &OutgoingMessage{Text: msg}
}

func ErrorMessage(err error) *OutgoingMessage {
	return NewMessage(fmt.Sprintf("ERROR: %s", err))
}

func (b *Bot) SendMessage(m *OutgoingMessage) (err error) {
	url := fmt.Sprintf(UrlTemplate, b.Subdomain, b.Token)

	out, err := json.Marshal(&m)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(out))
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Unexpected response code %d", resp.StatusCode))
	}

	if b.Debug {
		log.Printf("[slack] Sent message: %+v", m)
	}

	return nil
}

func (b *Bot) SlackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not supported", http.StatusBadRequest)
		log.Printf("WARN: Got a %s request to slack handler.", r.Method)
		return
	}

	var message IncomingMessage
	decoder := form.NewDecoder(r.Body)
	if err := decoder.Decode(&message); err != nil {
		http.Error(w, "Invalid post body", http.StatusBadRequest)
		log.Print("WARN: Could not decode slack message: ", err)
		return
	}

	// ignore messages comming from self
	if message.UserId == "USLACKBOT" {
		return
	}

	if b.Debug {
		log.Printf("[slack] Got message: %+v", message)
	}

	if b.MessageHandler != nil {
		response, err := b.MessageHandler(&message)
		if err != nil {
			response = ErrorMessage(err)
		}
		if response != nil {
			w.Header().Set("Content-Type", "application/json")
			encoder := json.NewEncoder(w)
			if err := encoder.Encode(&response); err != nil {
				panic(err)
			}
			if b.Debug {
				log.Printf("[slack] Sent response: %+v", response)
			}
		}
	}
}
