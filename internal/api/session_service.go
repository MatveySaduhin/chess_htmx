package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)
type SessionService struct {
    client    *mongo.Client
    db        *mongo.Database
    secretKey []byte
}

type Session struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    UserID    primitive.ObjectID `bson:"user_id"`
    Token     string             `bson:"token"`
    ExpiresAt time.Time          `bson:"expires_at"`
    CreatedAt time.Time          `bson:"created_at"`
}

func NewSessionService(client *mongo.Client, dbName string, secretKey string) *SessionService {
    return &SessionService{
        client: client,
        db: client.Database(dbName),
        secretKey: []byte(secretKey),
    }
}

func (ss *SessionService) CreateSession(userID primitive.ObjectID) (*Session, error) {
    token, err := generateRandomToken(32)
    if err != nil {
        return nil, err
    }

    session := &Session{
        UserID: userID,
        Token: token,
        ExpiresAt: time.Now().Add(24*time.Hour),
        CreatedAt: time.Now(),
    }

    collection := ss.db.Collection("sessions")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := collection.InsertOne(ctx, session)
    if err != nil {
        return nil, err
    }

    session.ID = result.InsertedID.(primitive.ObjectID)
    return session, nil
}

func (ss *SessionService) ValidateSession(token string) (*Session, error) {
    collection := ss.db.Collection("sessions")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var session Session
    err := collection.FindOne(ctx, bson.M{
        "token": token,
        "expires_at": bson.M{"$gt": time.Now()},
    }).Decode(&session)
    if err != nil {
        return nil, err
    }

    return &session, nil
}

func (ss *SessionService) DeleteSession(token string) error {
    collection := ss.db.Collection("sessions")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := collection.DeleteOne(ctx, bson.M{"token": token})
    return err
}

func generateRandomToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}
