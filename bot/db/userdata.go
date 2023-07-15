package db

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/go-redis/redis/v8"
)

type UserDataDB interface {
	Get(id snowflake.ID) (*UserData, error)
	Set(id snowflake.ID, data *UserData) error
	Del(id snowflake.ID) error
}

type userDataDBImpl struct {
	db *redis.Client
}

func (self *userDataDBImpl) Get(id snowflake.ID) (*UserData, error) {
	res := self.db.HGet(context.TODO(), "user-data", id.String())
	if err := res.Err(); err != nil {
		if err != redis.Nil {
			return nil, err
		} else {
			return NewUserData(id)
		}
	}
	var u *UserData = &UserData{}
	if err := json.Unmarshal([]byte(res.Val()), u); err != nil {
		return nil, err
	}
	return u, nil
}

func (self *userDataDBImpl) Set(id snowflake.ID, data *UserData) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res := self.db.HSet(context.TODO(), "user-data", id.String(), buf)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func (self *userDataDBImpl) Del(id snowflake.ID) error {
	res := self.db.HDel(context.TODO(), "user-data", id.String())
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}

func NewUserData(id snowflake.ID) (*UserData, error) {
	return &UserData{
		ID:          id,
		CreatedAt:   time.Now(),
		DataVersion: 0,
	}, nil
}

const UserDataVersion = 0

type UserData struct {
	ID snowflake.ID `json:"id"`

	CreatedAt time.Time    `json:"created_at"`
	BirthDay  [2]int       `json:"birth_day"`
	Location  UserLocation `json:"location"`

	LastMessageTime    time.Time     `json:"last_message_time"`
	MessageCount       int64         `json:"message_count"`
	GlobalLevel        UserDataLevel `json:"global_level"`
	GlobalMessageLevel UserDataLevel `json:"global_message_level"`
	GlobalVoiceLevel   UserDataLevel `json:"global_voice_level"`

	DataVersion int `json:"data_version"`
}

func (u *UserData) UnmarshalJSON(b []byte) error {
	type userData UserData
	var v struct {
		userData
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*u = UserData(v.userData)
	if !u.isValid() {
		if err := u.validate(b); err != nil {
			return err
		}
	}
	return nil
}

func (u *UserData) isValid() bool {
	return u.DataVersion < UserDataVersion
}

func (u *UserData) validate(b []byte) error {
	switch u.DataVersion {
	case UserDataVersion:
		return nil
	default:
		v, err := NewUserData(u.ID)
		if err != nil {
			return err
		}
		*u = *v
		return nil
	}
}

func NewUserLocation(str string) (UserLocation, error) {
	tl, err := time.LoadLocation(str)
	if err != nil {
		return UserLocation{time.UTC}, err
	}
	return UserLocation{tl}, nil
}

type UserLocation struct {
	*time.Location
}

func (u UserLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Location.String())
}

func (u *UserLocation) UnmarshalJSON(b []byte) error {
	var data string
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	lc, err := time.LoadLocation(data)
	if err != nil {
		return err
	}
	u.Location = lc
	return nil
}

type UserDataLevel struct {
	Point uint64 `json:"point"`
}

var i, a float64 = 10, 2

func (UserDataLevel) sum_required_level_point(n float64) float64 {
	return i * ((math.Pow(a, n) - 1) / (a - 1))
}

func (UserDataLevel) required_level_point(n float64) float64 {
	return i * math.Pow(float64(a), float64(n))
}

func (u UserDataLevel) ReqPoint() uint64 {
	return uint64(u.required_level_point(float64(u.Level())))
}

func (u UserDataLevel) SumReqPoint() uint64 {
	return uint64(u.sum_required_level_point(float64(u.Level())))
}

func (u UserDataLevel) Level() uint64 {
	for k := 0; k < 60; k++ {
		lv := u.sum_required_level_point(float64(k))
		if lv > float64(u.Point) {
			return uint64(k)
		}
	}
	return 0
}

func (u *UserDataLevel) Add(i uint64) {
	u.Point += i
}

func (u *UserDataLevel) AddRandom() {
	u.Add(uint64(rand.Int63n(10)) + 5)
}
