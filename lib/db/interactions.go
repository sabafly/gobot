package db

import (
	"context"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
)

type InteractionsDB interface {
	Get(id snowflake.ID) (string, error)
	Set(id snowflake.ID, poll string) error
	Remove(id snowflake.ID) error
}

type interactionsImpl struct {
	db *redis.Client
}

func (i *interactionsImpl) Get(id snowflake.ID) (string, error) {
	res := i.db.Get(context.TODO(), "interactions-"+id.String())
	if err := res.Err(); err != nil {
		return "", err
	}
	rt, err := res.Result()
	if err != nil {
		return "", err
	}
	return rt, nil
}

func (i *interactionsImpl) Set(id snowflake.ID, token string) error {
	res := i.db.Set(context.TODO(), "interactions-"+id.String(), token, time.Minute*14)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (i *interactionsImpl) Remove(id snowflake.ID) error {
	res := i.db.Del(context.TODO(), "interactions-"+id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}
