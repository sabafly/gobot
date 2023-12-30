package db

import (
	"encoding/json"
	"math/big"
	"math/rand"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type UserData struct {
	ID snowflake.ID `json:"id"`

	CreatedAt time.Time      `json:"created_at"`
	BirthDay  [2]int         `json:"birth_day"`
	Location  DataLocation   `json:"location"`
	Locale    discord.Locale `json:"locale"`

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
	return nil
}

type DataLocation struct {
	*time.Location
}

func (u *DataLocation) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Location.String())
}

func (u *DataLocation) UnmarshalJSON(b []byte) error {
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

func (UserDataLevel) sumRequiredLevelPoint(n *big.Int) *big.Int {
	n.Add(n, big.NewInt(3))
	return new(big.Int).Add(new(big.Int).Mul(i, new(big.Int).Div(new(big.Int).Sub(new(big.Int).Exp(a, n, nil), big.NewInt(1)), new(big.Int).Sub(a, big.NewInt(1)))), big.NewInt(0))
}

func (UserDataLevel) requiredLevelPoint(n *big.Int) *big.Int {
	n.Add(n, big.NewInt(3))
	return new(big.Int).Add(new(big.Int).Mul(i, new(big.Int).Exp(a, n, nil)), big.NewInt(0))
}

func (u UserDataLevel) ReqPoint() *big.Int {
	return u.requiredLevelPoint(u.Level())
}

func (u UserDataLevel) SumReqPoint() *big.Int {
	return u.sumRequiredLevelPoint(u.Level())
}

func (u UserDataLevel) Level() *big.Int {
	if u.Point == nil {
		u.Point = big.NewInt(0)
	}
	for k := 0; k < 999; k++ {
		lv := u.sumRequiredLevelPoint(big.NewInt(int64(k)))
		if lv.Cmp(u.Point) == 1 {
			return big.NewInt(int64(k))
		}
	}
	return big.NewInt(0)
}

func (u *UserDataLevel) Add(i *big.Int) {
	u.Point.Add(u.Point, i)
}

func (u *UserDataLevel) AddRandom() {
	r := rand.Intn(10)
	u.Add(new(big.Int).Add(big.NewInt(int64(r)), big.NewInt(15)))
}
