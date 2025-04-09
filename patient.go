package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoURI         = "mongodb+srv://dyno_reema_kumari_pharmeasy_8yq57:V0LaoCBfSPeftCs7@teleconsultation-oms-st.y7n1e.mongodb.net/?retryWrites=true&w=majority&tls=true"
	dbName           = "rx"
	patientCollName  = "patient"
	rxCollName       = "rx"
	tokenizerBaseURL = "https://dev-tokenizer.dev.pharmeasy.in"
	rxServiceBaseUrl = "https://qa-rx-service.qa.pharmeasy.in"
)

type PatientDoc struct {
	ID         primitive.ObjectID `bson:"_id"`
	TenantID   string             `bson:"tenantId"`
	CustomerID string             `bson:"customerId"`
	Name       string             `bson:"name"`
	HashedName string             `bson:"hashedName"`
	Gender     string             `bson:"gender"`
}

type RxDoc struct {
	ID         primitive.ObjectID `bson:"_id"`
	CustomerID string             `bson:"customerId"`
	TenantID   string             `bson:"tenantId"`
}

type RxPatient struct {
	Name   string `json:"name"`
	Gender string `json:"gender"`
}

type Origin struct {
	Source     string                 `json:"source"`
	Platform   string                 `json:"platform"`
	Attributes map[string]interface{} `json:"attributes"`
}

type UpdateRequestBody struct {
	Patient RxPatient `json:"patient"`
	Origin  Origin    `json:"origin"`
}

func main() {
	ctx := context.TODO()

	// MongoDB Connection
	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	defer client.Disconnect(ctx)

	patientColl := client.Database(dbName).Collection(patientCollName)
	rxColl := client.Database(dbName).Collection(rxCollName)

	// Clients for tokenizer and Rx service
	tokenizerClient := Client{
		httpClient: NewClient(),
		config: TokenizerService{
			Identifier: "ALLOY",
			BaseURL:    tokenizerBaseURL,
		},
	}

	rxServiceClient := RxClient{
		httpClient: NewClient(),
		config: RxService{
			BaseURL: rxServiceBaseUrl,
			Origin: Origin{
				Source:   "VENDOR",
				Platform: "TELECONSULTATION",
				Attributes: map[string]interface{}{
					"vendorId": "MeraDoc",
				},
			},
		},
	}

	// Filter patient docs by creation range
	start := time.Date(2025, 1, 26, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 4, 3, 23, 59, 59, 999000000, time.UTC)

	filter := bson.M{
		"createdAt": bson.M{"$gte": start, "$lte": end},
	}

	cursor, err := patientColl.Find(ctx, filter)
	if err != nil {
		log.Fatal("Patient Find error:", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var patient PatientDoc
		if err := cursor.Decode(&patient); err != nil {
			log.Println("Decode patient error:", err)
			continue
		}

		patientIDStr := patient.ID.Hex()
		log.Printf("Encrypted patient name for patientID %s: %s\n", patientIDStr, patient.Name)

		// Match Rx document using string ID
		rxFilter := bson.M{
			"patient.patientId": patientIDStr,
		}

		var rx RxDoc
		err := rxColl.FindOne(ctx, rxFilter).Decode(&rx)
		if err != nil {
			log.Printf("No Rx found for patientID %s: %v\n", patientIDStr, err)
			continue
		}

		rxID := rx.ID.Hex()

		// Step 1: Decrypt token once
		decryptReq1 := DecryptRequest{{Token: patient.Name}}
		resp1, err := tokenizerClient.Decrypt(ctx, &decryptReq1)
		if err != nil || len(resp1.Data) == 0 {
			log.Printf("First decryption failed for RxID %s: %v\n", rxID, err)
			continue
		}
		firstDecrypted := resp1.Data[0].Content

		// Step 2: Decrypt again
		decryptReq2 := DecryptRequest{{Token: firstDecrypted}}
		resp2, err := tokenizerClient.Decrypt(ctx, &decryptReq2)
		if err != nil || len(resp2.Data) == 0 {
			log.Printf("Second decryption failed for RxID %s: %v\n", rxID, err)
			continue
		}
		plainText := resp2.Data[0].Content

		log.Printf("Final plain name for RxID %s: %s\n", rxID, plainText)

		// Construct update payload
		body := UpdateRequestBody{
			Patient: RxPatient{
				Name:   plainText,
				Gender: patient.Gender,
			},
			Origin: rxServiceClient.config.Origin,
		}

		// Call Rx update API
		response, err := rxServiceClient.UpdatePatient(ctx, rxID, rx.TenantID, rx.CustomerID, body)
		if err != nil {
			log.Printf("Rx service update failed for RxID %s: %v\nResponse: %s\n", rxID, err, string(response))
			continue
		}

		log.Printf("Rx service update succeeded for RxID %s:\n%s\n", rxID, string(response))
	}

	if err := cursor.Err(); err != nil {
		log.Println("Cursor error:", err)
	}
}
