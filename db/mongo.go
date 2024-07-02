package db

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var Client *mongo.Client
var PersonCollection *mongo.Collection

type Person struct {
	ID        string `bson:"_id,omitempty"`
	Name      string `bson:"name"`
	ImagePath string `bson:"image_path"`
}

func InitDB() {
	var err error
	clientOptions := options.Client().ApplyURI("mongodb+srv://apifestanafestasana:ioocB57zKTcAPqaP@festa.1cgyqvm.mongodb.net/?retryWrites=true&w=majority&appName=festa")

	Client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		os.Exit(1)
	}

	err = Client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		fmt.Println("Failed to ping database:", err)
		os.Exit(1)
	}

	PersonCollection = Client.Database("testdb").Collection("persons")
}

func AddPerson(person *Person) (*mongo.InsertOneResult, error) {
	return PersonCollection.InsertOne(context.Background(), person)
}

func GetPersonByID(id string) (Person, error) {
	var person Person
	idd, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println(id, idd, err)

	}
	fmt.Println(id, idd)
	err = PersonCollection.FindOne(context.Background(), bson.M{"_id": idd}).Decode(&person)
	return person, err
}

func UpdatePerson(id string, update bson.M) (*mongo.UpdateResult, error) {
	return PersonCollection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": update})
}

func GetPersonAll() ([]Person, error) {
	var persons []Person

	// Encontre todos os documentos na coleção PersonCollection
	cursor, err := PersonCollection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	// Itere sobre o cursor e decodifique cada documento em uma struct Person
	for cursor.Next(context.Background()) {
		var person Person
		if err = cursor.Decode(&person); err != nil {
			return nil, err
		}
		persons = append(persons, person)
	}

	// Verifique se houve algum erro durante a iteração
	if err = cursor.Err(); err != nil {
		return nil, err
	}

	return persons, nil
}
