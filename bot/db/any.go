package db

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type Data[ID snowflake.ID | uuid.UUID] interface {
	ID() ID
}

type AnyDB[T Data[U], U snowflake.ID | uuid.UUID] interface {
	Get(context.Context, U) (*Result[T, U], error)
	Set(context.Context, U, T) error
	Del(context.Context, U) error
	GetAll(context.Context) (*Results[T, U], error)
}

type config struct {
	Timeout *time.Duration
}

type OptionFunc func(*config) *config

func WithTimeout(timeout time.Duration) OptionFunc {
	return func(c *config) *config {
		c.Timeout = &timeout
		return c
	}
}

func newAnyDB[T Data[U], U snowflake.ID | uuid.UUID](db *redis.Client, opt ...OptionFunc) *anyDB[T, U] {
	cfg := new(config)
	for _, of := range opt {
		cfg = of(cfg)
	}
	return &anyDB[T, U]{
		db:     db,
		mus:    make(map[U]*sync.Mutex),
		config: *cfg,
	}
}

type anyDB[T Data[U], U snowflake.ID | uuid.UUID] struct {
	db  *redis.Client
	mu  sync.Mutex
	mus map[U]*sync.Mutex

	config config
}

func (a *anyDB[T, U]) Mu(id U) *sync.Mutex {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mus[id] == nil {
		a.mus[id] = new(sync.Mutex)
	}
	return a.mus[id]
}

func (a *anyDB[T, U]) Get(ctx context.Context, id U) (*Result[T, U], error) {
	a.Mu(id).Lock()
	var res *redis.StringCmd
	switch a.config.Timeout {
	case nil:
		res = a.db.HGet(ctx, toChainCase(reflect.TypeOf(*new(T)).Name()), reflect.ValueOf(id).MethodByName("String").Call(nil)[0].String())
	default:
		res = a.db.Get(ctx, fmt.Sprintf("%s-%s", toChainCase(reflect.TypeOf(*new(T)).Name()), reflect.ValueOf(id).MethodByName("String").Call(nil)[0].String()))
	}
	if err := res.Err(); err != nil {
		return nil, err
	}
	data := new(T)
	if err := json.Unmarshal([]byte(res.Val()), data); err != nil {
		return nil, err
	}
	return &Result[T, U]{
		data: *data,
		closer: func() error {
			a.Mu(id).Unlock()
			return nil
		},
	}, nil
}

func (a *anyDB[T, U]) Set(ctx context.Context, id U, data T) error {
	a.Mu(id).Lock()
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	switch a.config.Timeout {
	case nil:
		res := a.db.HSet(ctx, toChainCase(reflect.TypeOf(*new(T)).Name()), reflect.ValueOf(id).MethodByName("String").Call(nil)[0].String(), buf)
		if err := res.Err(); err != nil {
			return err
		}
	default:
		res := a.db.Set(ctx, fmt.Sprintf("%s-%s", toChainCase(reflect.TypeOf(*new(T)).Name()), reflect.ValueOf(id).MethodByName("String").Call(nil)[0].String()), buf, *a.config.Timeout)
		if err := res.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (a *anyDB[T, U]) Del(ctx context.Context, id U) error {
	a.Mu(id).Lock()
	switch a.config.Timeout {
	case nil:
		res := a.db.HDel(ctx, toChainCase(reflect.TypeOf(*new(T)).Name()), reflect.ValueOf(id).MethodByName("String").Call(nil)[0].String())
		if err := res.Err(); err != nil {
			return err
		}
	default:
		res := a.db.Del(ctx, fmt.Sprintf("%s-%s", toChainCase(reflect.TypeOf(*new(T)).Name()), reflect.ValueOf(id).MethodByName("String").Call(nil)[0].String()))
		if err := res.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (a *anyDB[T, U]) GetAll(ctx context.Context) (*Results[T, U], error) {
	var res *redis.StringStringMapCmd
	switch a.config.Timeout {
	case nil:
		res := a.db.HGetAll(ctx, toChainCase(reflect.TypeOf(*new(T)).Name()))
		if err := res.Err(); err != nil {
			return nil, err
		}
	}
	data := make([]T, 0)
	closers := make([]U, 0)
	for k, v := range res.Val() {
		d := new(T)
		if err := json.Unmarshal([]byte(v), d); err != nil {
			return nil, err
		}
		data = append(data, *d)
		key := new(U)
		if err := json.Unmarshal([]byte(k), key); err != nil {
			return nil, err
		}
		a.Mu(*key).Lock()
		closers = append(closers, *key)
	}
	return &Results[T, U]{
		data: data,
		closer: func() error {
			for _, v := range closers {
				a.Mu(v).Unlock()
			}
			return nil
		},
	}, nil
}

type Result[T Data[U], U snowflake.ID | uuid.UUID] struct {
	data   T
	closer func() error
}

func (r Result[T, U]) Value() T {
	return r.data
}

func (r Result[T, U]) Close() error {
	return r.closer()
}

type Results[T Data[U], U snowflake.ID | uuid.UUID] struct {
	data   []T
	closer func() error
}

func (r Results[T, U]) Value() []T {
	return r.data
}

func (r Results[T, U]) Close() error {
	return r.closer()
}

func toChainCase(s string) string {
	b := &strings.Builder{}
	for i, r := range s {
		if i == 0 {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		if unicode.IsUpper(r) {
			b.WriteRune('-')
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
