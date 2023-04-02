package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
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
	return Calc{
		Input:     "0",
		LastInput: "0",
		id:        uuid.New(),
		Mode:      CalcModeInput,
	}
}

type Calc struct {
	id               uuid.UUID
	Input            string
	inputFloat       float64
	LastInput        string
	lastInputFloat   float64
	currentIndicator CalcIndicator
	Mode             CalcMode
	answer           float64
}

type CalcMode int

const (
	CalcModeInput       CalcMode = 1 // 最初の入力 値が一つのみ
	CalcModeInputSecond CalcMode = 2 // 二つ目の入力 値が二つ
	CalcModeAnswer      CalcMode = 3 // 計算後の状態
	CalcModeTemp        CalcMode = 4 // 演算子入力後の状態
)

type CalcIndicator rune

const (
	CalcIndicatorNone     CalcIndicator = '_'
	CalcIndicatorPlus     CalcIndicator = '+'
	CalcIndicatorMinus    CalcIndicator = '-'
	CalcIndicatorMultiple CalcIndicator = '×'
	CalcIndicatorDivide   CalcIndicator = '÷'
	CalcIndicatorPercent  CalcIndicator = '%'
)

func (c Calc) MarshalJSON() ([]byte, error) {
	alias := struct {
		ID               uuid.UUID     `json:"id"`
		Input            string        `json:"input"`
		LastInput        string        `json:"last_input"`
		CurrentIndicator CalcIndicator `json:"current_indicator"`
		Mode             CalcMode      `json:"mode"`
		Answer           float64       `json:"answer"`
	}{
		ID:               c.id,
		Input:            c.Input,
		LastInput:        c.LastInput,
		CurrentIndicator: c.currentIndicator,
		Mode:             c.Mode,
		Answer:           c.answer,
	}
	return json.Marshal(alias)
}

func (c *Calc) UnmarshalJSON(b []byte) error {
	alias := struct {
		ID               uuid.UUID     `json:"id"`
		Input            string        `json:"input"`
		LastInput        string        `json:"last_input"`
		CurrentIndicator CalcIndicator `json:"current_indicator"`
		Mode             CalcMode      `json:"mode"`
		Answer           float64       `json:"answer"`
	}{}
	err := json.Unmarshal(b, &alias)
	if err != nil {
		return err
	}
	*c = Calc{
		id:               alias.ID,
		Input:            alias.Input,
		LastInput:        alias.LastInput,
		currentIndicator: alias.CurrentIndicator,
		Mode:             alias.Mode,
		answer:           alias.Answer,
	}
	return nil
}

func (c Calc) ID() uuid.UUID {
	return c.id
}

func (c Calc) Indicator() CalcIndicator {
	return c.currentIndicator
}

func (c *Calc) InputDo() {
	switch c.Mode {
	case CalcModeTemp:
		c.Mode = CalcModeInputSecond
		c.Input = ""
	case CalcModeAnswer:
		c.Reset()
	}
}

func (c *Calc) Back() {
	switch c.Mode {
	case CalcModeAnswer:
		c.lastInputFloat = 0
		c.inputFloat = c.answer
		c.answer = 0
		c.toString()
		c.Mode = CalcModeInput
	default:
		if strings.Count(c.Input, "e") != 0 {
			return
		}
		if len(c.Input) <= 1 || (strings.HasPrefix(c.Input, "-") && len(c.Input) <= 2) {
			c.Input = "0"
		} else {
			c.Input = c.Input[:len(c.Input)-1]
		}
	}
}

func (c *Calc) CE() {
	switch c.Mode {
	case CalcModeTemp:
		c.InputDo()
		c.Input += "0"
	case CalcModeAnswer:
		c.Reset()
	default:
		c.inputFloat = 0
		c.toString()
	}
}

func (c *Calc) Reset() {
	*c = Calc{
		id:        c.id,
		Input:     "0",
		LastInput: "0",
		Mode:      CalcModeInput,
	}
}

func (c *Calc) beforeIndicator() {
	switch c.Mode {
	case CalcModeInputSecond:
		if err := c.do(); err != nil {
			panic(err)
		}
		c.lastInputFloat = c.answer
		c.toString()
		c.Mode = CalcModeTemp
	case CalcModeAnswer:
		c.lastInputFloat = c.answer
		c.toString()
		c.Mode = CalcModeTemp
	}
}

func (c *Calc) Plus() {
	c.beforeIndicator()
	c.currentIndicator = CalcIndicatorPlus
	if c.Mode != CalcModeTemp {
		c.LastInput = c.Input
	}
	c.Mode = CalcModeTemp
}

func (c *Calc) Minus() {
	c.beforeIndicator()
	c.currentIndicator = CalcIndicatorMinus
	if c.Mode != CalcModeTemp {
		c.LastInput = c.Input
	}
	c.Mode = CalcModeTemp
}

func (c *Calc) Multiple() {
	c.beforeIndicator()
	c.currentIndicator = CalcIndicatorMultiple
	if c.Mode != CalcModeTemp {
		c.LastInput = c.Input
	}
	c.Mode = CalcModeTemp
}

func (c *Calc) Divide() {
	c.beforeIndicator()
	c.currentIndicator = CalcIndicatorDivide
	if c.Mode != CalcModeTemp {
		c.LastInput = c.Input
	}
	c.Mode = CalcModeTemp
}

func (c *Calc) Do() {
	if c.Mode == CalcModeAnswer {
		if err := c.toFloat(); err != nil {
			panic(err)
		}
		c.lastInputFloat = c.answer
		c.toString()
		c.Mode = CalcModeInputSecond
		if err := c.do(); err != nil {
			panic(err)
		}
		c.Mode = CalcModeAnswer
	} else {
		if err := c.do(); err != nil {
			panic(err)
		}
		c.Mode = CalcModeAnswer
	}
}

func (c *Calc) toFloat() error {
	var err error
	if c.LastInput == "" {
		c.LastInput = "0"
	}
	c.lastInputFloat, err = strconv.ParseFloat(c.LastInput, 64)
	if err != nil {
		return err
	}
	if c.Input == "" {
		c.Input = "0"
	}
	c.inputFloat, err = strconv.ParseFloat(c.Input, 64)
	if err != nil {
		return err
	}
	return nil
}

func (c *Calc) toString() {
	c.LastInput = fmt.Sprintf("%v", c.lastInputFloat)
	c.Input = fmt.Sprintf("%v", c.inputFloat)
}

func (c *Calc) do() error {
	err := c.toFloat()
	if err != nil {
		return err
	}
	switch c.currentIndicator {
	case CalcIndicatorPlus:
		c.answer = c.lastInputFloat + c.inputFloat
	case CalcIndicatorMinus:
		c.answer = c.lastInputFloat - c.inputFloat
	case CalcIndicatorMultiple:
		c.answer = c.lastInputFloat * c.inputFloat
	case CalcIndicatorDivide:
		c.answer = c.lastInputFloat / c.inputFloat
	}
	c.toString()
	return nil
}

func (c *Calc) Message(formatter func([]discord.Embed) []discord.Embed) (discord.MessageCreate, error) {
	mes := discord.MessageCreate{}
	err := c.toFloat()
	if err != nil {
		return mes, err
	}
	var content string
	switch c.Mode {
	case CalcModeInput:
		content = fmt.Sprintf("```\r\r%32v```", c.Input)
	case CalcModeInputSecond:
		content = fmt.Sprintf("```\r%v %s\r%32v```", c.lastInputFloat, string(c.currentIndicator), c.Input)
	case CalcModeAnswer:
		if c.currentIndicator == 0 {
			content = fmt.Sprintf("```\r%v =\r%32v```", c.inputFloat, c.inputFloat)
		} else {
			content = fmt.Sprintf("```\r%v %s %v =\r%32v```", c.lastInputFloat, string(c.currentIndicator), c.inputFloat, c.answer)
		}
	case CalcModeTemp:
		content = fmt.Sprintf("```\r%v %s\r%32v```", c.lastInputFloat, string(c.currentIndicator), c.lastInputFloat)
	}
	mes.Content = content
	mes.Components = c.Component()
	return mes, nil
}

func (c Calc) Component() []discord.ContainerComponent {
	return []discord.ContainerComponent{
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:utilcalc:ce:%s", c.id.String()),
				Label:    "CE",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleDanger),
				CustomID: fmt.Sprintf("handler:utilcalc:c:%s", c.id.String()),
				Label:    "C",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:utilcalc:back:%s", c.id.String()),
				Label:    "←",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:utilcalc:act:divide:%s", c.id.String()),
				Label:    "÷",
			},
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 7, c.id.String()),
				Label:    "7",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 8, c.id.String()),
				Label:    "8",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 9, c.id.String()),
				Label:    "9",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:utilcalc:act:multiple:%s", c.id.String()),
				Label:    "×",
			},
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 4, c.id.String()),
				Label:    "4",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 5, c.id.String()),
				Label:    "5",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 6, c.id.String()),
				Label:    "6",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:utilcalc:act:minus:%s", c.id.String()),
				Label:    "-",
			},
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 1, c.id.String()),
				Label:    "1",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 2, c.id.String()),
				Label:    "2",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 3, c.id.String()),
				Label:    "3",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStylePrimary),
				CustomID: fmt.Sprintf("handler:utilcalc:act:plus:%s", c.id.String()),
				Label:    "+",
			},
		},
		discord.ActionRowComponent{
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%s:%s", "±", c.id.String()),
				Label:    "±",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%d:%s", 0, c.id.String()),
				Label:    "0",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSecondary),
				CustomID: fmt.Sprintf("handler:utilcalc:num:%s:%s", ".", c.id.String()),
				Label:    ".",
			},
			discord.ButtonComponent{
				Style:    discord.ButtonStyle(discord.ButtonStyleSuccess),
				CustomID: fmt.Sprintf("handler:utilcalc:do:%s", c.id.String()),
				Label:    "=",
			},
		},
	}
}
