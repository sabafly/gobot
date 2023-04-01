package db

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type InteractionsDB interface {
	Get(id uuid.UUID) (string, error)
	Set(id uuid.UUID, token string) error
	Remove(id uuid.UUID) error
}

type interactionsImpl struct {
	db *redis.Client
}

func (i *interactionsImpl) Get(id uuid.UUID) (string, error) {
	res := i.db.Get(context.TODO(), "interactions"+id.String())
	if err := res.Err(); err != nil {
		return "", err
	}
	rt, err := res.Result()
	if err != nil {
		return "", err
	}
	return rt, nil
}

func (i *interactionsImpl) Set(id uuid.UUID, token string) error {
	res := i.db.Set(context.TODO(), "interactions"+id.String(), token, time.Minute*14)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (i *interactionsImpl) Remove(id uuid.UUID) error {
	res := i.db.Del(context.TODO(), "interactions"+id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}
