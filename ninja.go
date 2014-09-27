package main

import (
	"encoding/json"
	"fmt"
	"github.com/ajg/form"
	"log"
	"net/http"
	"ninja/slack"
	"os"
	"strings"
)

type SlackResponse struct {
	Channel     string `json:"channel,omitempty"`
	From        string `json:"username,omitempty"`
	Message     string `json:"text"`
	UseMarkdown bool   `json:"mrkdwn,omitempty"`
}

// func sendMessage(m *SlackResponse) *error {

// }

func handleSlackMessage(m *slack.SlackMessage) *SlackResponse {
	var response SlackResponse

	command := strings.Trim(m.Text, " \t")
	switch command {
	case "foo":
		response.Message = "foo"
		log.Println("GOT FOO!")
	}

	return &response
}

func slackHandler(w http.ResponseWriter, r *http.Request) {
	/* Handle incomming slack messages */
	var message slack.SlackMessage

	if r.Method != "POST" {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Invalid method: %s", r.Method)
		return
	}

	decoder := form.NewDecoder(r.Body)
	if err := decoder.Decode(&message); err != nil {
		http.Error(w, "Invalid post body", http.StatusBadRequest)
		log.Print("Error decoding slack message: ", err)
		return
	}

	log.Printf("Got message: %+v", message)

	response := handleSlackMessage(&message)
	if response != nil {
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(&response); err != nil {
			panic(err)
		}
		log.Printf("Response: %+v", response)
	}

	// decoder := json.NewDecoder(r.Body)
	// var payload slack_message
	// err := decoder.Decode(&payload)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// log.Printf("%s %s", r.Method, r.RequestURI)
	// fmt.Fprintf(w, "Keke! foo, I love %s!", r.URL.Path[1:])
}

func main() {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	//x := http.Client()

	http.HandleFunc("/slack", slackHandler)

	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
