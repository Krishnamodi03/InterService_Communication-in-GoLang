package main

import (
	"context"
	"fmt"
	"log"
	// "time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	// "go.mongodb.org/mongo-driver/mongo/readpref"
)

type manager struct {
	Connection *mongo.Client
	Ctx        context.Context
	Cancel     context.CancelFunc
}

var Mgr manager

func connectDb() {
	// Connect to MongoDB
	mongoURI := "mongodb://localhost:27017"

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(Mgr.Ctx, clientOptions)
	// Check the error
	if err != nil {
		log.Fatal(err)
	}
	// Set the client to the manager
	Mgr.Connection = client
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// err = client.Connect(ctx)
	// // Check the error
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// err = client.Ping(ctx, readpref.Primary())
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// Mgr = manager{Connection: client, Ctx: ctx, Cancel: cancel}
	fmt.Println("Database Connected...!!!")
}
func Close(client *mongo.Client, ctx context.Context, cancel context.CancelFunc) {
	// Close the connection
	defer cancel()
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}
func main() {
	connectDb()
}
