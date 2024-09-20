package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Collection
var Ctx context.Context

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	// Load MongoDB URL from environment variable
	mongoURI := os.Getenv("MONGODB_URL")
	if mongoURI == "" {
		log.Fatal("MONGODB_URL environment variable is not set")
	}

	clientOptions := options.Client().ApplyURI(mongoURI)

	// Create a context with a timeout and make sure to call cancel to avoid a context leak
	Ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Ensures the context resources are released

	// Connect to MongoDB
	client, err := mongo.Connect(Ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(Ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Set the database and collection variables
	DB = client.Database("user").Collection("user_info")

}
