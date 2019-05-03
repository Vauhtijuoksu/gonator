package main

import (
	"encoding/json"
	"fmt"
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

func main() {
	http.HandleFunc("/donations", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
		}

		var donations Donations
		for {
			fetchDonations := getDonations("https://vauhtijuoksu.otit.fi/api/donations")
			var newDonations Donations
			for _, donation := range fetchDonations {
				if inList(donation, donations) == false {
					donations = append(donations, donation)
					newDonations = append(newDonations, donation)
				}
			}
			// Write message to browser
			if err := conn.WriteJSON(newDonations); err != nil {
				fmt.Println(err)
			}

			time.Sleep(10 * time.Second)
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
