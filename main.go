package main

import (
	"context"
	_ "encoding/json"
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
	collectionName   = "rx"
	tokenizerBaseURL = "https://dev-tokenizer.dev.pharmeasy.in"
	rxServiceBaseUrl = "https://qa-rx-service.qa.pharmeasy.in"
)

type Patient struct {
	ID         primitive.ObjectID     `bson:"_id"`
	TenantID   string                 `bson:"tenantId"`
	CustomerID string                 `bson:"customerId"`
	CreatedAt  time.Time              `bson:"createdAt"`
	Origin     map[string]interface{} `bson:"origin"`
	Patient    PatientData            `bson:"patient"`
}

type PatientData struct {
	PatientID  string `bson:"patientId"`
	Name       string `bson:"name"`
	HashedName string `bson:"hashedName"`
	Gender     string `bson:"gender"`
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

	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Println("MongoDB disconnect error:", err)
		}
	}()

	collection := client.Database(dbName).Collection(collectionName)

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

	start := time.Date(2025, 1, 26, 10, 26, 52, 860*1e6, time.UTC)
	end := time.Date(2025, 4, 3, 23, 59, 59, 999*1e6, time.UTC)

	filter := bson.M{
		"createdAt":                  bson.M{"$gte": start, "$lte": end},
		"origin.attributes.vendorId": "MeraDoc",
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatal("Find error:", err)
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	for cursor.Next(ctx) {
		var patient Patient
		if err := cursor.Decode(&patient); err != nil {
			log.Println("Decode error:", err)
			continue
		}

		rxId := patient.ID.Hex()
		customerId := patient.CustomerID
		tenantId := patient.TenantID
		log.Printf("Processing RxID: %s | CustomerID: %s | TenantID: %s\n", rxId, customerId, tenantId)

		// Step 1: Decrypt once
		decryptReq1 := DecryptRequest{{Token: patient.Patient.Name}}
		resp1, err := tokenizerClient.Decrypt(ctx, &decryptReq1)
		if err != nil || len(resp1.Data) == 0 {
			log.Println("First decryption failed for", rxId, "error:", err)
			continue
		}
		firstDecrypted := resp1.Data[0].Content

		// Step 2: Decrypt again
		decryptReq2 := DecryptRequest{{Token: firstDecrypted}}
		resp2, err := tokenizerClient.Decrypt(ctx, &decryptReq2)
		if err != nil || len(resp2.Data) == 0 {
			log.Println("Second decryption failed for", rxId, "error:", err)
			continue
		}
		plainText := resp2.Data[0].Content

		log.Printf("Final plain name for RxID %s: %s\n", rxId, plainText)

		// Step 3: Call Rx service to update
		body := UpdateRequestBody{
			Patient: RxPatient{
				Name:   plainText,
				Gender: patient.Patient.Gender,
			},
			Origin: Origin{
				Source:   "VENDOR",
				Platform: "TELECONSULTATION",
				Attributes: map[string]interface{}{
					"vendorId": "MeraDoc",
				},
			},
		}

		response, err := rxServiceClient.UpdatePatient(ctx, rxId, tenantId, customerId, body)

		if err != nil {
			log.Printf("Rx service update failed for RxID %s: %v\nResponse: %s\n", rxId, err, string(response))
			continue
		}

		log.Printf("Rx service update succeeded for RxID %s:\n%s\n", rxId, string(response))

	}

}
