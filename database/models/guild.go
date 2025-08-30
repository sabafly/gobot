package models

import "github.com/disgoorg/snowflake/v2"

type Guild struct {
	ID snowflake.ID `gorm:"primary_key;column:id;type:bigint(20) unsigned;not null"`
}
