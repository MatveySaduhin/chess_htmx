package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Profile struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AuthUserID      string             `bson:"auth_user_id" json:"auth_user_id"`
	DisplayName     string             `bson:"display_name" json:"display_name"`
	RapidRating     int                `bson:"rapid_rating" json:"rapid_rating"`
	BlitzRating     int                `bson:"blitz_rating" json:"blitz_rating"`
	BulletRating    int                `bson:"bullet_rating" json:"bullet_rating"`
	ClassicalRating int                `bson:"classical_rating" json:"classical_rating"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

type ProfileService struct {
	collection *mongo.Collection
}

func NewProfileService(client *mongo.Client, dbName string) (*ProfileService, error) {
	service := &ProfileService{
		collection: client.Database(dbName).Collection("profiles"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := service.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "auth_user_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("profiles_auth_user_id_unique"),
	})
	if err != nil {
		return nil, fmt.Errorf("create profile auth_user_id index: %w", err)
	}

	return service, nil
}

func (s *ProfileService) FindOrCreateByAuthUserID(ctx context.Context, authUserID, displayName string) (*Profile, error) {
	authUserID = strings.TrimSpace(authUserID)
	if authUserID == "" {
		return nil, fmt.Errorf("auth_user_id is required")
	}

	now := time.Now().UTC()
	if strings.TrimSpace(displayName) == "" {
		displayName = defaultDisplayName(authUserID)
	}

	filter := bson.M{"auth_user_id": authUserID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"auth_user_id":     authUserID,
			"display_name":     displayName,
			"rapid_rating":     1200,
			"blitz_rating":     1200,
			"bullet_rating":    1200,
			"classical_rating": 1200,
			"created_at":       now,
			"updated_at":       now,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var profile Profile
	err := s.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&profile)
	if err != nil {
		return nil, fmt.Errorf("find or create profile: %w", err)
	}

	return &profile, nil
}

func defaultDisplayName(authUserID string) string {
	if len(authUserID) <= 8 {
		return "Player " + authUserID
	}
	return "Player " + authUserID[:8]
}
