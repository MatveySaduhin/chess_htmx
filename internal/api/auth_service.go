package api

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
    client *mongo.Client
    db     *mongo.Database
}

type User struct {
    ID       primitive.ObjectID `bson:"_id,omitempty"`
    Name     string             `bson:"name"`
    Email    string             `bson:"email"`
    Password string             `bson:"password"`
}

func NewAuthService(client *mongo.Client, dbName string) (*AuthService, error) {
    db := client.Database(dbName)

    return &AuthService{
        client: client,
        db:     db,
    }, nil
}

func (as *AuthService) Close() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    as.client.Disconnect(ctx)
}

func (as *AuthService) Authenticate(email, password string) (*User, error) {
    var user User
    collection := as.db.Collection("users")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, fmt.Errorf("user not found")
        }
        return nil, err
    }

    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    if err != nil {
        return nil, fmt.Errorf("invalid password")
    }

    return &user, nil
}

func (as AuthService) NewUser(name, email, password string) (*User, error) {
    if err := validate(email, password); err != nil {
        return nil, err
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }

    user := &User {
        Name: name,
        Email: email,
        Password: string(hashedPassword),
    }

    collection := as.db.Collection("users")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var existing User
    err = collection.FindOne(ctx, bson.M{"email": email}).Decode(&existing)
    if err == nil {
        return nil, fmt.Errorf("user with this email alreadt exists")
    }

    result, err := collection.InsertOne(ctx, user)
    if err != nil {
        return nil, err
    }

    user.ID = result.InsertedID.(primitive.ObjectID)
    return user, nil
}

func validate(email, password string) error {
    if email == "" {
        return fmt.Errorf("email is required")
    }
    if len(password) < 6 {
        return fmt.Errorf("password must be at least 6 characters")
    }
    return nil
}
