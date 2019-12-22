package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
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

type Hashtag struct {
	Name   string  `bson:"nameHashtag"`
	Amount float32 `bson:"amount"`
}

type Hashtags []Hashtag

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

func parseHashtags(message string, amount float32) Hashtags {
	var hashtagsName []string
	for _, word := range strings.Split(message, " ") {
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			hashtagsName = append(hashtagsName, word)
		}
	}

	hashtagAmount := amount / float32(len(hashtagsName))

	var hashtags Hashtags
	for _, hashtagName := range hashtagsName {
		var hashtag Hashtag
		hashtag.Name = hashtagName
		hashtag.Amount = hashtagAmount
	}

	return hashtags
}

func apiPoll(collection *mongo.Collection, hashtagCollection *mongo.Collection) {
	for {
		fetchDonations := getDonations("http://192.168.43.155:5000/donates")

		for _, donation := range fetchDonations {

			var hashtags Hashtags
			var result Donation
			filter := bson.D{{"donationId", donation.DonationId}}
			err := collection.FindOne(context.TODO(), filter).Decode(&result)
			if err != nil {
				if err.Error() == "mongo: no documents in result" {
					_, err := collection.InsertOne(context.TODO(), donation)
					if err != nil {
						log.Fatal(err)
					}

					if strings.Contains(donation.Message, "#") {
						hashtags = parseHashtags(donation.Message, donation.Amount)
					}
				} else {
					log.Fatal(err)
				}
			} else if result.Name == "Anonyymi" && result.Message == "" {
				update := bson.D{
					{"$set", bson.D{{"message", donation.Message}}},
					{"$set", bson.D{{"nameDonator", donation.Name}}},
				}
				_, err := collection.UpdateOne(context.TODO(), filter, update)
				if err != nil {
					log.Fatal(err)
				}
				if strings.Contains(donation.Message, "#") {
					hashtags = parseHashtags(donation.Message, donation.Amount)
				}
			}

			if len(hashtags) != 0 {
				for _, hashtag := range hashtags {
					filter := bson.D{{"hashtag", hashtag.Name}}
					err := collection.FindOne(context.TODO(), filter).Decode(&result)
					if err != nil {
						if err.Error() == "mongo: no documents in result" {
							fmt.Println(hashtag)
							_, _ = hashtagCollection.InsertOne(context.TODO(), hashtag)
						} else {
							log.Fatal(err)
						}
					}
					update := bson.D{
						{"$set", bson.D{{"hashtag", hashtag.Name}}},
						{"$inc", bson.D{{"amount", hashtag.Amount}}},
					}
					fmt.Println(hashtag)
					_, err = collection.UpdateOne(context.TODO(), filter, update)
					if err != nil {
						log.Fatal(err)
					}
				}
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
	hashtagCollection := client.Database("gonator").Collection("hashtags")
	go apiPoll(collection, hashtagCollection)

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
			donationMessage.OperationType = "insert"

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

	http.ListenAndServe(":8080", nil)

}
