package poll

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Vauhtijuoksu/gonator/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func Poll(ctx context.Context, collection *mongo.Collection, url string) {
	for {

		fetchDonations := helpers.GetDonations(url)

		for _, donation := range fetchDonations {

			var result helpers.Donation
			filter := bson.D{{Key: "donationId", Value: donation.DonationID}}
			err := collection.FindOne(ctx, filter).Decode(&result)
			if err != nil {
				if err.Error() == "mongo: no documents in result" {
					insertResult, err := collection.InsertOne(context.TODO(), donation)
					fmt.Println("Inserted document: ", insertResult.InsertedID)
					if err != nil {
						log.Println(err)
					}
				} else {
					log.Println(err)
				}
			} else if result.Name == "Anonyymi" && result.Message == "" {
				update := bson.D{
					{Key: "$set", Value: bson.D{{Key: "message", Value: donation.Message}}},
					{Key: "$set", Value: bson.D{{Key: "nameDonator", Value: donation.Name}}},
				}
				updateResult, err := collection.UpdateOne(ctx, filter, update)
				if err != nil {
					log.Println(err)
				}

				fmt.Printf("Matched %v documents and updated %v documents.\n", updateResult.MatchedCount, updateResult.ModifiedCount)
			}
		}
		time.Sleep(10 * time.Second)
	}
}
