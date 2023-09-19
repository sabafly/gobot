package db

import (
	"encoding/json"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/internal/flags"
)

func NewUsedID(id snowflake.ID, t IDType, ref *snowflake.ID, flag UsedFlags) UsedID {
	return UsedID{
		id:        id,
		Type:      t,
		Ref:       ref,
		UsedFlags: flag,
	}
}

type UsedID struct {
	id        snowflake.ID
	Type      IDType        `json:"type"`
	Ref       *snowflake.ID `json:"ref"`
	UsedFlags UsedFlags     `json:"used_flags"`
}

type usedID struct {
	UsedID
	ID snowflake.ID `json:"id"`
}

func (u *UsedID) UnmarshalJSON(b []byte) error {
	v := usedID{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*u = v.UsedID
	u.id = v.ID
	return nil
}

func (u UsedID) MarshalJSON() ([]byte, error) {
	return json.Marshal(usedID{
		UsedID: u,
		ID:     u.id,
	})
}

func (u UsedID) ID() snowflake.ID {
	return u.id
}

type IDType int

const (
	IDTypeGuild = iota
	IDTypeChannel
	IDTypeMessage
	IDTypeMember
)

type UsedFlags int

const (
	UsedFlagTicket UsedFlags = 1 << iota
)

func (f UsedFlags) Add(bits ...UsedFlags) UsedFlags {
	return flags.Add(f, bits...)
}

func (f UsedFlags) Remove(bits ...UsedFlags) UsedFlags {
	return flags.Remove(f, bits...)
}

func (f UsedFlags) Has(bits ...UsedFlags) bool {
	return flags.Has(f, bits...)
}

func (f UsedFlags) Missing(bits ...UsedFlags) bool {
	return flags.Missing(f, bits...)
}
