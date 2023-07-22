package db

import (
	"context"
	"encoding/json"
	"math/big"
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
	Point *big.Int `json:"point"`
}

var i = big.NewInt(10)
var a = big.NewInt(2)

func (UserDataLevel) sum_required_level_point(n *big.Int) *big.Int {
	n.Add(n, big.NewInt(5))
	return new(big.Int).Add(new(big.Int).Mul(i, new(big.Int).Div(new(big.Int).Sub(new(big.Int).Exp(a, n, nil), big.NewInt(1)), new(big.Int).Sub(a, big.NewInt(1)))), big.NewInt(50))
}

func (UserDataLevel) required_level_point(n *big.Int) *big.Int {
	n.Add(n, big.NewInt(5))
	return new(big.Int).Add(new(big.Int).Mul(i, new(big.Int).Exp(a, n, nil)), big.NewInt(50))
}

func (u UserDataLevel) ReqPoint() *big.Int {
	return u.required_level_point(u.Level())
}

func (u UserDataLevel) SumReqPoint() *big.Int {
	return u.sum_required_level_point(u.Level())
}

func (u UserDataLevel) Level() *big.Int {
	for k := 0; k < 999; k++ {
		lv := u.sum_required_level_point(big.NewInt(int64(k)))
		if lv.Cmp(u.Point) == 1 {
			return big.NewInt(int64(k))
		}
	}
	return nil
}

func (u *UserDataLevel) Add(i *big.Int) {
	u.Point.Add(u.Point, i)
}

func (u *UserDataLevel) AddRandom() {
	r := rand.Intn(10)
	u.Add(new(big.Int).Add(big.NewInt(int64(r)), big.NewInt(15)))
}
