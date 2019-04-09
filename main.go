package main

import (
	"net/http"
    "io/ioutil"
    "time"
    "encoding/json"

	"github.com/gorilla/websocket"
)

type Donation struct {
    DonationId          string  `json"DonationId"`
    Name                string  `json"Name"`
    Amount              float32 `json"Amount"`
    Message             string  `json"Message"`
    MessageAnswer       string  `json"Message"`
    CollectorImageUrl   string  `json"CollectorImageUrl"`
    CurrencySymbol      string  `json"CurrencySymbol"`
    CollectionUrl       string  `json"CollaectionUrl"`
    TransactionDate     string  `json"TransactionDate"`
}

type Donations []Donation


var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	http.HandleFunc("/donations", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)

        response, _ := http.Get("https://oma.kummit.fi/f/Donation/GetDonations/?collectionId=COL-16-1229")

        responseData, _ := ioutil.ReadAll(response.Body)

        var donations Donations

        json.Unmarshal(responseData, &donations)

        if err := conn.WriteJSON(donations); err != nil {
            return
        }
		for {
            time.Sleep(2 * time.Second)
			// Write message to browser
            if err := conn.WriteJSON(donations); err != nil {
				return
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe(":8080", nil)
}
