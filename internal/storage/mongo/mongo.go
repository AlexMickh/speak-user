package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-user/internal/config"
	"github.com/AlexMickh/speak-user/internal/domain/models"
	mongouuid "github.com/AlexMickh/speak-user/pkg/utils/mongo-uuid"
	"github.com/AlexMickh/speak-user/pkg/utils/retry"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Storage struct {
	client *mongo.Client
	coll   *mongo.Collection
}

func New(ctx context.Context, cfg config.DBConfig) (*Storage, error) {
	const op = "storage.mongo.New"

	var client *mongo.Client
	var coll *mongo.Collection
	connString := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=admin", cfg.User, cfg.Password, cfg.Host, cfg.Port)

	err := retry.WithDelay(5, 500*time.Millisecond, func() error {
		var err error

		client, err = mongo.Connect(options.Client().ApplyURI(connString).SetRegistry(mongouuid.MongoRegistry))
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		err = client.Ping(ctx, readpref.Primary())
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		coll = client.Database(cfg.Database).Collection(cfg.Collection)

		_, err = coll.Indexes().CreateOne(
			ctx,
			mongo.IndexModel{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		client: client,
		coll:   coll,
	}, nil
}

func (s *Storage) Close(ctx context.Context) {
	if err := s.client.Disconnect(ctx); err != nil {
		panic(err)
	}
}

func (s *Storage) SaveUser(ctx context.Context, user models.User) error {
	const op = "storage.mongo.SaveUser"

	_, err := s.coll.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetUser(ctx context.Context, email string) (models.User, error) {
	const op = "storage.mongo.GetUser"

	var user models.User
	err := s.coll.FindOne(ctx, bson.D{{Key: "email", Value: email}}).Decode(&user)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) ChangeEmailVerified(ctx context.Context, id uuid.UUID) error {
	const op = "storage.mongo.ChangeEmailVerified"

	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "is_email_verified", Value: true},
		}},
	}

	_, err := s.coll.UpdateByID(ctx, id, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	username *string,
	description *string,
	profileImageUrl *string,
) (models.User, error) {
	const op = "storage.mongo.UpdateUser"

	data := struct {
		Username        *string `bson:"username,omitempty" json:"username,omitempty"`
		Description     *string `bson:"description,omitempty" json:"description,omitempty"`
		ProfileImageUrl *string `bson:"profile_image_url,omitempty" json:"profile_image_url,omitempty"`
	}{
		Username:        username,
		Description:     description,
		ProfileImageUrl: profileImageUrl,
	}
	update := bson.D{
		{Key: "$set", Value: data},
	}

	_, err := s.coll.UpdateByID(ctx, id, update)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	var user models.User
	err = s.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&user)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) DeleteUser(ctx context.Context, id uuid.UUID) (string, error) {
	const op = "storage.mongo.DeleteUser"

	var user models.User
	err := s.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&user)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	_, err = s.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return *user.ProfileImageUrl, nil
}
