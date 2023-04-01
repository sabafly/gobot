package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type CalcDB interface {
	Get(uuid.UUID) (Calc, error)
	Set(uuid.UUID, Calc) error
	Remove(uuid.UUID) error
}

type CalcDBImpl struct {
	db *redis.Client
}

func (c *CalcDBImpl) Get(id uuid.UUID) (Calc, error) {
	res := c.db.Get(context.TODO(), "calc"+id.String())
	if err := res.Err(); err != nil {
		return Calc{}, err
	}
	buf := []byte(res.Val())
	val := Calc{}
	err := json.Unmarshal(buf, &val)
	if err != nil {
		return Calc{}, err
	}
	return val, nil
}

func (c *CalcDBImpl) Set(id uuid.UUID, data Calc) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := c.db.Set(context.TODO(), "calc"+id.String(), buf, time.Minute*14)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (c *CalcDBImpl) Remove(id uuid.UUID) error {
	res := c.db.Del(context.TODO(), id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewCalc() Calc {
	return Calc{}
}

type Calc struct {
	Input            string        `json:"input"`
	InputFloat       float64       `json:"-"`
	LastInput        string        `json:"last_input"`
	LastInputFloat   float64       `json:"-"`
	CurrentIndicator CalcIndicator `json:"current_indicator"`
}

type CalcIndicator rune

const (
	CalcIndicatorPlus     CalcIndicator = '+'
	CalcIndicatorMinus    CalcIndicator = '-'
	CalcIndicatorMultiple CalcIndicator = '×'
	CalcIndicatorDivide   CalcIndicator = '÷'
	CalcIndicatorPercent  CalcIndicator = '%'
)
