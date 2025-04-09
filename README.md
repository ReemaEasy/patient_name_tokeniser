# RX Collection Updater

This Go script is used to update encrypted patient information in the `rx` collection of a MongoDB database and sync the updated data with an external Rx service.

## Features

- Decrypts the `patient.name` field using a custom tokenizer client.
- Sends to update patient details to the external Rx service using the RxClient.
- Fully configurable MongoDB URL.

## Prerequisites

- Go installed (1.18+ recommended)
- Access credentials/config for:
  - Tokenizer Client (`Client`)
  - Rx Service Client (`RxClient`)

## Configuration

Update the MongoDB connection URL before running the script.
use command to run the script : go run main.go 
In `main.go`, modify the MongoDB URL:
```go
mongoURI := "<your-mongo-db-url>"
