package main

import (
	"github.com/rakyll/coop"
	"log"
	"time"
)

func UpdateStuff() {
	log.Println("Update stuff")
}

func Tasks() {
	coop.Every(time.Minute, UpdateStuff)
}
