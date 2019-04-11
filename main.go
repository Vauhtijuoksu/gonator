package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Donation struct {
	DonationId        string  `json"DonationId"`
	Name              string  `json"Name"`
	Amount            float32 `json"Amount"`
	Message           string  `json"Message"`
	MessageAnswer     string  `json"Message"`
	CollectorImageUrl string  `json"CollectorImageUrl"`
	CurrencySymbol    string  `json"CurrencySymbol"`
	CollectionUrl     string  `json"CollaectionUrl"`
	TransactionDate   string  `json"TransactionDate"`
}

type Donations []Donation

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func getDonations(url string) Donations {
	var donations Donations

	response, _ := http.Get(url)
	responseData, _ := ioutil.ReadAll(response.Body)
	json.Unmarshal(responseData, &donations)

	return donations
}

func main() {
	http.HandleFunc("/donations", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)

		var donationCount int
		for {
			donations := getDonations("https://vauhtijuoksu.otit.fi/api/donations")
			if len(donations) > donationCount {
				newDonations := donations[0 : len(donations)-donationCount]
				// Write message to browser
				if err := conn.WriteJSON(newDonations); err != nil {
					return
				}
			}
			donationCount = len(donations)
			time.Sleep(10 * time.Second)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe(":8080", nil)
}
