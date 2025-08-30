package play

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
)

type BetData struct {
	id         uuid.UUID
	userID     snowflake.ID
	userBet    map[snowflake.ID]int64
	userSelect map[snowflake.ID]int
	selects    []string
}
