package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Vauhtijuoksu/gonator/helpers"
	"github.com/Vauhtijuoksu/gonator/poll"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DonationMessage struct {
	OperationType string           `bson:"operationType"`
	Donation      helpers.Donation `bson:"fullDocument"`
}

type UpdateWebsocket struct {
	Donations []DonationMessage `json:"Donations"`
}

var (
	collection *mongo.Collection
	ctx        context.Context

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func main() {

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongodb:27017"))
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, nil)
	ctx = context.Background()

	if err != nil {
		log.Println(err)
	}

	log.Println("Connected to MongoDB")

	collection = client.Database("gonator").Collection("donations")
	go poll.Poll(ctx, collection, "https://potti.mieli.fi/f/Donation/GetDonations/?collectionId=COL-16-2330&pageSize=50&startAt=0", 10)

	http.HandleFunc("/", index)
	http.HandleFunc("/donations", donations)
	http.HandleFunc("/goal", goal)
	http.HandleFunc("/getDonations", getDonations)
	http.HandleFunc("/bar", bar)

	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/html/index.html")
}

func donations(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
	}

	// Get all donations and send them first.
	cur, err := collection.Find(ctx, bson.D{{}}, options.Find())
	if err != nil {
		log.Println(err)
	}

	var updateWebsocket UpdateWebsocket

	for cur.Next(ctx) {
		var donation helpers.Donation
		var donationMessage DonationMessage

		err := cur.Decode(&donation)
		if err != nil {
			log.Println(err)
		}

		donationMessage.Donation = donation
		donationMessage.OperationType = "firstInsert"

		updateWebsocket.Donations = append(updateWebsocket.Donations, donationMessage)

	}

	if err := conn.WriteJSON(updateWebsocket); err != nil {
		log.Println(err)
	}

	// Watch if there are changes at mongodb, if there are, send them.
	cs, err := collection.Watch(ctx, mongo.Pipeline{}, options.ChangeStream().SetFullDocument(options.UpdateLookup))
	if err != nil {
		log.Println(err)
	}

	for cs.Next(ctx) {
		var updateWebsocket UpdateWebsocket
		var donationMessage DonationMessage

		err := cs.Decode(&donationMessage)
		if err != nil {
			log.Println(err)
		}
		updateWebsocket.Donations = append(updateWebsocket.Donations, donationMessage)
		if err := conn.WriteJSON(updateWebsocket); err != nil {
			log.Println(err)
		}
	}
}

func goal(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	var goal int
	for {
		fetchgoal := helpers.GetGoal("https://vauhtijuoksu.fi/api/goal")
		if fetchgoal != goal {
			goal = fetchgoal
			if err := conn.WriteJSON(goal); err != nil {
				log.Println(err)
			}
		}
		time.Sleep(60 * time.Second)
	}
}

func getDonations(w http.ResponseWriter, r *http.Request) {
	cur, err := collection.Find(ctx, bson.D{{}}, options.Find())
	if err != nil {
		log.Println(err)
	}
	var donations []helpers.Donation
	if err = cur.All(ctx, &donations); err != nil {
		log.Println(err)
	}
	e, err := json.Marshal(donations)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(e)
}

func bar(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/html/bar.html")
}
