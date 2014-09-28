package main

import (
	"bitbucket.org/ckvist/twilio/twiml"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"time"
)

func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Ninja up for %s", time.Since(Env.Started))
}

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	f := r.URL.Path[1:]
	log.Debugf("Serving %s", f)
	http.ServeFile(w, r, f)
}

func CallHandler(w http.ResponseWriter, r *http.Request) {
	log.Infof("Incomming call from %s, %s", r.RemoteAddr, r.UserAgent())
	resp := twiml.NewResponse()
	resp.Action(twiml.Play{Url: Env.Vars.AppURL + "/assets/roll.mp3"})
	resp.Send(w)
}

func SetupWeb() {
	log.Infof("Starting webserver on port %s", Env.Vars.ServerPort)

	http.HandleFunc("/", DefaultHandler)
	http.HandleFunc("/slack", Env.Bot.SlackHandler)
	http.HandleFunc("/assets/", StaticHandler)
	http.HandleFunc("/call", CallHandler)

	log.Fatal(http.ListenAndServe(":"+Env.Vars.ServerPort, nil))
}
