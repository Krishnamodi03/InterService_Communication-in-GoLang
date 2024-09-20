package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type manager struct {
	Connection *mongo.Client
	Ctx        context.Context
	Cancel     context.CancelFunc
}

var Mgr manager

func connectDb() {
	// Connect to MongoDB
	mongoURI := "mongodb+srv://admin:admin@cluster0.x4pzq.mongodb.net/"
	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(Mgr.Ctx, clientOptions)
	// Check the error
	if err != nil {
		log.Fatal(err)
	}
	// Set the client to the manager
	Mgr.Connection = client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	Mgr.Ctx, Mgr.Cancel = ctx, cancel

	fmt.Println("Database Connected...!!!")
	defer Close(Mgr.Connection, Mgr.Ctx, Mgr.Cancel)
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
