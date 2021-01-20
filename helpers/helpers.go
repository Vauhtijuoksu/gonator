package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

func GetFromAPI(url string) []byte {

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

func GetGoal(url string) int {
	var goal int

	responseData := GetFromAPI(url)
	json.Unmarshal(responseData, &goal)

	return goal

}

func GetDonations(url string) Donations {
	var donations Donations

	responseData := GetFromAPI(url)
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
