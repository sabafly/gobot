package xppoint

import (
	"math/big"
	"math/rand/v2"
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
	case level < 31:
		x = 5*n - 38
	case 31 <= level:
		x = 9*n - 158
	}
	x *= multiplier
	return x
}

// 次の方程式を使用して、レベルに到達するまでにどれだけの経験値が収集されたかを決定できます。
func TotalPoint(level int64) uint64 {
	f := (&big.Float{}).SetInt64(level)
	x := big.NewFloat(0)
	switch {
	case level <= 16: // level^2 + 6 × level
		x.Add((&big.Float{}).Mul(f, f), (&big.Float{}).Mul(big.NewFloat(6), f))
	case level <= 31: // 2.5 × level^2 – 40.5 × level + 360
		x.Add(
			(&big.Float{}).Mul(big.NewFloat(2.5), (&big.Float{}).Mul(f, f)),
			(&big.Float{}).Mul(big.NewFloat(-40.5), f),
		).Add(x, big.NewFloat(360))
	case 32 <= level: // 4.5 × level^2 – 162.5 × level + 2220
		x.Add(
			(&big.Float{}).Mul(big.NewFloat(4.5), (&big.Float{}).Mul(f, f)),
			(&big.Float{}).Mul(big.NewFloat(-162.5), f),
		).Add(x, big.NewFloat(2220))
	}
	x.Mul(x, big.NewFloat(multiplier))
	u, _ := x.Uint64()
	return u
}

func level(points uint64) int64 {
	points /= multiplier
	f := (&big.Float{}).SetUint64(points)
	x := big.NewFloat(0)
	switch {
	case points <= 352:
		x.Add(
			(&big.Float{}).Sqrt((&big.Float{}).Add(f, big.NewFloat(9))),
			big.NewFloat(-3),
		)
	case points <= 1507:
		x.Add(
			big.NewFloat(81.0/10.0),
			(&big.Float{}).Sqrt(
				(&big.Float{}).Mul(
					big.NewFloat(2.0/5.0),
					(&big.Float{}).Add(
						f,
						big.NewFloat(-7839.0/40.0)),
				),
			),
		)
	case 1508 <= points:
		x.Add(
			big.NewFloat(325.0/18.0),
			(&big.Float{}).Sqrt(
				(&big.Float{}).Mul(
					big.NewFloat(2.0/9.0),
					(&big.Float{}).Add(
						f,
						big.NewFloat(-54215.0/72.0),
					),
				),
			),
		)
	}
	i, _ := x.Int64()
	return i
}
