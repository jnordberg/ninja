package main

import (
	log "github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net/url"
	"time"
)

type User struct {
	Id         bson.ObjectId `bson:"_id,omitempty"`
	UserId     string        `bson:"user_id"`
	Name       string        `bson:"name"`
	Phone      string        `bson:"phone"`
	PhoneValid bool          `bson:"phone_valid"`
	PhoneCode  string        `bson:"phone_code"`
	Runner     bool          `bson:"runner"`
}

type Item struct {
	Name      string        `bson:"name"`
	OwnerId   bson.ObjectId `bson:"owner_id"`
	OwnerName string        `bson:"owner_name"`
}

type Run struct {
	Id      bson.ObjectId `bson:"_id,omitempty"`
	Runner  bson.ObjectId `bson:"runner"`
	Items   []Item        `bson:"items"`
	Started time.Time     `bson:"started"`
}

func GetCollection(name string) *mgo.Collection {
	session := Env.DBSession //.Clone()
	collection := session.DB(Env.Vars.MongoDB).C(name)
	return collection
}

func SetupDatabase() {
	log.SetFormatter(&log.TextFormatter{ForceColors: Env.Vars.ForceColors})

	u, err := url.Parse(Env.Vars.MongoURL)
	if err != nil {
		log.Panic(err)
	}

	log.Infof("Connectiong to database %s", u.Host)

	session, err := mgo.Dial(Env.Vars.MongoURL)
	if err != nil {
		log.Panic(err)
	}

	Env.DBSession = session

	log.Infof("Connected to database (%s)", Env.Vars.MongoDB)
}
