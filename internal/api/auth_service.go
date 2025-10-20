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
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	Name          string             `bson:"name"`
	Email         string             `bson:"email"`
	Password      string             `bson:"password"`
	Role          string             `bson:"role"`
	CreatedAt     time.Time          `bson:"created_at"`
	FormattedDate string             `bson:"-"`
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

func (as AuthService) NewUser(name, email, password, role string) (*User, error) {
	if err := validate(email, password); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		Name:      name,
		Email:     email,
		Password:  string(hashedPassword),
		Role:      role,
		CreatedAt: time.Now(),
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

func (as AuthService) RenameUser(userID string, newName string) error {
	collection := as.db.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert string ID to ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	// Update the user's name
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"name": newName}},
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (as *AuthService) GetUserByID(userID primitive.ObjectID) (*User, error) {
	collection := as.db.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (as *AuthService) GetAllUsers() ([]User, error) {
	collection := as.db.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all documents in the collection using an empty filter
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	// Important: close the cursor to free resources
	defer cursor.Close(ctx)

	var users []User
	// Iterate through the cursor and decode each document into a User struct
	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, user)
	}

	// Check for any errors that occurred during cursor iteration
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor iteration error: %w", err)
	}

	return users, nil
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
