package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Donation struct {
	DonationID        string  `json:"DonationId" bson:"donationId"`
	Name              string  `json:"Name" bson:"nameDonator"`
	Amount            float32 `json:"Amount" bson:"amountMoney"`
	Message           string  `json:"Message" bson:"message"`
	MessageAnswer     string  `json:"MessageAnswer" bson:"messageAnswer"`
	CollectorImageURL string  `json:"CollectorImageUrl" bson:"collectorImageurl"`
	CurrencySymbol    string  `json:"CurrencySymbol" bson:"currencySymbol"`
	CollectionURL     string  `json:"CollaectionUrl" bson:"collectionUrl"`
	TransactionDate   string  `json:"TransactionDate" bson:"transactionDate"`
}

type Donations []Donation

type DonationMessage struct {
	OperationType string   `bson:"operationType"`
	Donation      Donation `bson:"fullDocument"`
}

type UpdateWebsocket struct {
	Donations []DonationMessage `json:"Donations"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func getFromAPI(url string) []byte {

	response, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}

	return responseData
}

func getGoal(url string) int {
	var goal int

	responseData := getFromAPI(url)
	json.Unmarshal(responseData, &goal)

	return goal

}

func getDonations(url string) Donations {
	var donations Donations

	responseData := getFromAPI(url)
	json.Unmarshal(responseData, &donations)

	return donations
}

func inList(donation Donation, donations Donations) bool {

	for _, iterDonation := range donations {
		if donation == iterDonation {
			return true
		}
	}

	return false

}

func apiPoll(ctx context.Context, collection *mongo.Collection) {
	for {

		fetchDonations := getDonations("https://vauhtijuoksu.otit.fi/api/donations")

		for _, donation := range fetchDonations {

			var result Donation
			filter := bson.D{{Key: "donationId", Value: donation.DonationID}}
			err := collection.FindOne(ctx, filter).Decode(&result)
			if err != nil {
				if err.Error() == "mongo: no documents in result" {
					insertResult, err := collection.InsertOne(context.TODO(), donation)
					fmt.Println("Inserted document: ", insertResult.InsertedID)
					if err != nil {
						log.Fatal(err)
					}
				} else {
					log.Fatal(err)
				}
			} else if result.Name == "Anonyymi" && result.Message == "" {
				update := bson.D{
					{Key: "$set", Value: bson.D{{Key: "message", Value: donation.Message}}},
					{Key: "$set", Value: bson.D{{Key: "nameDonator", Value: donation.Name}}},
				}
				updateResult, err := collection.UpdateOne(ctx, filter, update)
				if err != nil {
					log.Fatal(err)
				}

				fmt.Printf("Matched %v documents and updated %v documents.\n", updateResult.MatchedCount, updateResult.ModifiedCount)
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func main() {

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongo1:27017"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB")

	collection := client.Database("gonator").Collection("donations")
	go apiPoll(ctx, collection)

	http.HandleFunc("/donations", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
		}

		findOptions := options.Find()
		cur, err := collection.Find(ctx, bson.D{{}}, findOptions)
		if err != nil {
			log.Fatal(err)
		}

		var updateWebsocket UpdateWebsocket

		for cur.Next(ctx) {
			var donation Donation
			var donationMessage DonationMessage

			err := cur.Decode(&donation)
			if err != nil {
				log.Fatal(err)
			}

			donationMessage.Donation = donation
			donationMessage.OperationType = "firstInsert"

			updateWebsocket.Donations = append(updateWebsocket.Donations, donationMessage)

		}

		if err := conn.WriteJSON(updateWebsocket); err != nil {
			fmt.Println(err)
		}

		if err != nil {
			log.Fatal(err)
		}

		cs, err := collection.Watch(ctx, mongo.Pipeline{}, options.ChangeStream().SetFullDocument(options.UpdateLookup))
		if err != nil {
			log.Fatal(err)
		}

		for cs.Next(ctx) {
			// var donations Donations
			var updateWebsocket UpdateWebsocket
			var donationMessage DonationMessage

			err := cs.Decode(&donationMessage)
			if err != nil {
				log.Fatal(err)
			}
			updateWebsocket.Donations = append(updateWebsocket.Donations, donationMessage)
			fmt.Println(updateWebsocket)
			// Write message to browser
			if err := conn.WriteJSON(updateWebsocket); err != nil {
				fmt.Println(err)
			}
		}
	})

	http.HandleFunc("/goal", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
		}

		var goal int
		for {
			fetchgoal := getGoal("https://vauhtijuoksu.otit.fi/api/goal")
			if fetchgoal != goal {
				goal = fetchgoal
				// Write message to browser
				if err := conn.WriteJSON(goal); err != nil {
					fmt.Println(err)
				}
			}
			time.Sleep(60 * time.Second)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "bar.html")
	})

	http.HandleFunc("/getdonations", func(w http.ResponseWriter, r *http.Request) {
		findOptions := options.Find()
		cur, err := collection.Find(ctx, bson.D{{}}, findOptions)
		if err != nil {
			log.Fatal(err)
		}
		var donations []Donation
		if err = cur.All(ctx, &donations); err != nil {
			log.Fatal(err)
		}
		e, err := json.Marshal(donations)
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(e)
	})

	http.ListenAndServe(":8080", nil)
}
