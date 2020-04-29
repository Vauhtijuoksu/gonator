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
	DonationId        string  `json:"DonationId" bson:"donationId"`
	Name              string  `json:"Name" bson:"nameDonator"`
	Amount            float32 `json:"Amount" bson:"amountMoney"`
	Message           string  `json:"Message" bson:"message"`
	MessageAnswer     string  `json:"MessageAnswer" bson:"messageAnswer"`
	CollectorImageUrl string  `json:"CollectorImageUrl" bson:"collectorImageurl"`
	CurrencySymbol    string  `json:"CurrencySymbol" bson:"currencySymbol"`
	CollectionUrl     string  `json:"CollaectionUrl" bson:"collectionUrl"`
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

func getFromApi(url string) []byte {

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

	responseData := getFromApi(url)
	json.Unmarshal(responseData, &goal)

	return goal

}

func getDonations(url string) Donations {
	var donations Donations

	responseData := getFromApi(url)
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

func apiPoll(collection *mongo.Collection) {
	for {

		fetchDonations := getDonations("https://vauhtijuoksu.otit.fi/api/donations")

		for _, donation := range fetchDonations {

			var result Donation
			filter := bson.D{{"donationId", donation.DonationId}}
			err := collection.FindOne(context.TODO(), filter).Decode(&result)
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
					{"$set", bson.D{{"message", donation.Message}}},
					{"$set", bson.D{{"nameDonator", donation.Name}}},
				}
				updateResult, err := collection.UpdateOne(context.TODO(), filter, update)
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

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongo1:27017"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(context.TODO(), nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB")

	collection := client.Database("gonator").Collection("donations")
	go apiPoll(collection)

	http.HandleFunc("/donations", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
		}

		findOptions := options.Find()
		cur, err := collection.Find(context.TODO(), bson.D{{}}, findOptions)
		if err != nil {
			log.Fatal(err)
		}

		var updateWebsocket UpdateWebsocket

		for cur.Next(context.TODO()) {
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

		ctx := context.Background()
		clientWach, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongo1:27017"))
		if err != nil {
			log.Fatal(err)
		}
		collection := clientWach.Database("gonator").Collection("donations")

		cs, err := collection.Watch(ctx, mongo.Pipeline{}, options.ChangeStream().SetFullDocument(options.UpdateLookup))
		if err != nil {
			log.Fatal(err)
		}
		defer cs.Close(ctx)

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
		cur, err := collection.Find(context.TODO(), bson.D{{}}, findOptions)
		if err != nil {
			log.Fatal(err)
		}
		var donations []Donation
		if err = cur.All(ctx, &donations); err != nil {
			log.Fatal(err)
		}
		e, err := json.Marshal(donations)
		if err != nil {
			fmt.Println(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(e)
	})

	http.ListenAndServe(":8080", nil)

}
