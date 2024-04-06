package mongoClient

import (
	"context"
	"fmt"
	"log"

	"github.com/amirfaghihi/migrator/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	Client *mongo.Client
}

func (m *Mongo) InitClient() error {
	var err error
	clientOptions := options.Client().ApplyURI(config.GetMongoURI())
	m.Client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}
	return nil
}

func (m *Mongo) RunFindQuery(collectionName string, query bson.M) {
	// Use the client to interact with MongoDB
	collection := m.Client.Database("sumx").Collection(collectionName)

	// Find documents
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var user User
		err := cursor.Decode(&user)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found user: %+v\n", user)
	}

	if err := cursor.Err(); err != nil {
		log.Fatal(err)
	}
}
