package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User is the model that governs all notes objects retrived or inserted into the DB
type Order struct {
	Id         primitive.ObjectID `json:"id" bson:"_id"`
	UserID     primitive.ObjectID `json:"userId" bson:"userID" validate:"required"`
	Order_name *string            `json:"order_name" validate:"required,min=2,max=100"`
	Price      *string            `json:"price" validate:"required"`
	Created_at time.Time          `json:"created_at"`
	Updated_at time.Time          `json:"updated_at"`
}
