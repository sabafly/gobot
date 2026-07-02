package play

import (
	"fmt"
	"math/rand/v2"
)

var (
	randomCardFunc             = func() Card { return CardFromIndex(rand.N(54)) }
	randomCardWithoutJokerFunc = func() Card { return CardFromIndex(rand.N(52)) }
)

func RandomCard() Card {
	return randomCardFunc()
}

func RandomCardWithoutJoker() Card {
	return randomCardWithoutJokerFunc()
}

func CardFromIndex(index int) Card {
	if index < 0 || index > 53 {
		return CardNone
	}
	if index == 52 {
		return CardJokerBlack
	}
	if index == 53 {
		return CardJokerRed
	}
	suit := CardSuitSpades + (Card(index/13) << 4)
	number := Card(index%13 + 1)
	return suit | number
}

func NewCard(suit Card, number int) Card {
	if number < 1 || number > 13 {
		return CardNone
	}
	if suit.Suit() == CardSuitJoker {
		switch number {
		case 1:
			return CardJokerBlack
		case 2:
			return CardJokerRed
		default:
			return CardNone
		}
	}
	return suit | Card(number)
}

type Card uint8

const (
	CardNone Card = iota
	CardNum1
	CardNum2
	CardNum3
	CardNum4
	CardNum5
	CardNum6
	CardNum7
	CardNum8
	CardNum9
	CardNum10
	CardNumJack
	CardNumQueen
	CardNumKing
)

const (
	CardSuitSpades Card = iota << 4
	CardSuitHearts
	CardSuitDiamonds
	CardSuitClubs

	CardSuitJoker = 0xF0
)

const (
	CardJokerBlack Card = CardSuitJoker | iota + 1
	CardJokerRed
)

func (c Card) String() string {
	if c == CardNone {
		return "N/A"
	}
	suit := c.Suit()
	var suitStr string
	switch suit {
	case CardSuitSpades:
		suitStr = "♠️"
	case CardSuitHearts:
		suitStr = "♥️"
	case CardSuitDiamonds:
		suitStr = "♦️"
	case CardSuitClubs:
		suitStr = "♣️"
	case CardSuitJoker:
		switch c {
		case CardJokerBlack:
			return "🃏⬛"
		case CardJokerRed:
			return "🃏🟥"
		default:
			return fmt.Sprintf("🃏%s", c.NumStr())
		}
	default:
		return "Unknown Suit"
	}
	return fmt.Sprintf("%s%s", suitStr, c.NumStr())
}

func (c Card) Suit() Card {
	return c & 0xF0
}

func (c Card) Number() int {
	return int(c & 0x0F)
}

func (c Card) NumStr() string {
	switch c.Number() {
	case 1:
		return "A"
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	default:
		return fmt.Sprintf("%d", c.Number())
	}
}

func (c Card) Index() int {
	if !c.IsValid() {
		return -1
	}
	if c == CardJokerBlack {
		return 52
	}
	if c == CardJokerRed {
		return 53
	}
	return int(c.Suit()>>4)*13 + c.Number() - 1
}

func (c Card) IsJoker() bool {
	return c.Suit() == CardSuitJoker
}

func (c Card) IsValid() bool {
	if c.IsJoker() {
		return c == CardJokerBlack || c == CardJokerRed
	}
	return c.IsValidNumber()
}

func (c Card) IsValidNumber() bool {
	return c.Number() >= 1 && c.Number() <= 13
}
