package xppoint

import (
	"math"
	"math/big"
	"math/rand"
)

type XP uint64

func (xp XP) Level() int64 {
	return level(uint64(xp))
}

func (xp *XP) Add(n uint64) {
	*xp += XP(n)
}

func (xp *XP) AddRandom() uint64 {
	add := rand.Uint64()%16 + 15
	xp.Add(add)
	return add
}

const multiplier = 25

// 0から始まる
// 次のレベルに経験値がどれだけいるか
//
// level=0 -> 1レベルに上がるまでの経験値
func RequiredPoint(level int64) uint64 {
	n := uint64(level)
	var x uint64
	switch {
	case level < 16:
		x = 2*n + 7
	case 16 <= level && level < 31:
		x = 5*n - 38
	case 31 <= level:
		x = 9*n - 158
	}
	x *= multiplier
	return x
}

func RequiredPointTotal(level int64) uint64 {
	n := big.NewInt(level)
	x := new(big.Int)
	switch {
	case level < 16:
		x.Add((&big.Int{}).Exp(n, big.NewInt(2), nil), (&big.Int{}).Mul(big.NewInt(6), n))
	case 16 <= level && level < 31:
		x.Add(
			(&big.Int{}).Div((&big.Int{}).Mul(big.NewInt(5), (&big.Int{}).Exp(n, big.NewInt(2), nil)), big.NewInt(2)),
			(&big.Int{}).Div((&big.Int{}).Mul(big.NewInt(-81), n), big.NewInt(2)),
		).Add(x, big.NewInt(360))
	case 31 < level:
		x.Add(
			(&big.Int{}).Div((&big.Int{}).Mul(big.NewInt(9), (&big.Int{}).Exp(n, big.NewInt(2), nil)), big.NewInt(2)),
			(&big.Int{}).Div((&big.Int{}).Mul(big.NewInt(-325), n), big.NewInt(2)),
		).Add(x, big.NewInt(2200))
	}
	x.Mul(x, big.NewInt(multiplier))
	return uint64(x.Int64())
}

func level(points uint64) int64 {
	points /= multiplier
	var x float64
	switch {
	case points < 353:
		x = math.Sqrt(float64(points+9)) - 3
	case 353 <= points && points < 1508:
		x = math.Sqrt((2.0/5)*(float64(points)-(7839/40))) + (81 / 10)
	case 1508 <= points:
		x = math.Sqrt((2.0/9)*(float64(points)-(54215/72))) + (325 / 18)
	}
	return int64(x)
}
