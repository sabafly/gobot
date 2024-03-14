// Code generated by ent, DO NOT EDIT.

package member

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/sabafly/gobot/internal/permissions"
	"github.com/sabafly/gobot/internal/xppoint"
)

const (
	// Label holds the string label denoting the member type in the database.
	Label = "member"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldPermission holds the string denoting the permission field in the database.
	FieldPermission = "permission"
	// FieldXp holds the string denoting the xp field in the database.
	FieldXp = "xp"
	// FieldUserID holds the string denoting the user_id field in the database.
	FieldUserID = "user_id"
	// FieldLastXp holds the string denoting the last_xp field in the database.
	FieldLastXp = "last_xp"
	// FieldMessageCount holds the string denoting the message_count field in the database.
	FieldMessageCount = "message_count"
	// FieldLastNotifiedLevel holds the string denoting the last_notified_level field in the database.
	FieldLastNotifiedLevel = "last_notified_level"
	// EdgeGuild holds the string denoting the guild edge name in mutations.
	EdgeGuild = "guild"
	// EdgeUser holds the string denoting the user edge name in mutations.
	EdgeUser = "user"
	// Table holds the table name of the member in the database.
	Table = "members"
	// GuildTable is the table that holds the guild relation/edge.
	GuildTable = "members"
	// GuildInverseTable is the table name for the Guild entity.
	// It exists in this package in order to avoid circular dependency with the "guild" package.
	GuildInverseTable = "guilds"
	// GuildColumn is the table column denoting the guild relation/edge.
	GuildColumn = "guild_members"
	// UserTable is the table that holds the user relation/edge.
	UserTable = "members"
	// UserInverseTable is the table name for the User entity.
	// It exists in this package in order to avoid circular dependency with the "user" package.
	UserInverseTable = "users"
	// UserColumn is the table column denoting the user relation/edge.
	UserColumn = "user_id"
)

// Columns holds all SQL columns for member fields.
var Columns = []string{
	FieldID,
	FieldPermission,
	FieldXp,
	FieldUserID,
	FieldLastXp,
	FieldMessageCount,
	FieldLastNotifiedLevel,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "members"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"guild_members",
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for i := range ForeignKeys {
		if column == ForeignKeys[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultPermission holds the default value on creation for the "permission" field.
	DefaultPermission permissions.Permission
	// DefaultXp holds the default value on creation for the "xp" field.
	DefaultXp xppoint.XP
	// DefaultMessageCount holds the default value on creation for the "message_count" field.
	DefaultMessageCount uint64
	// DefaultLastNotifiedLevel holds the default value on creation for the "last_notified_level" field.
	DefaultLastNotifiedLevel uint64
)

// OrderOption defines the ordering options for the Member queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByXp orders the results by the xp field.
func ByXp(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldXp, opts...).ToFunc()
}

// ByUserID orders the results by the user_id field.
func ByUserID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUserID, opts...).ToFunc()
}

// ByLastXp orders the results by the last_xp field.
func ByLastXp(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLastXp, opts...).ToFunc()
}

// ByMessageCount orders the results by the message_count field.
func ByMessageCount(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMessageCount, opts...).ToFunc()
}

// ByLastNotifiedLevel orders the results by the last_notified_level field.
func ByLastNotifiedLevel(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLastNotifiedLevel, opts...).ToFunc()
}

// ByGuildField orders the results by guild field.
func ByGuildField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newGuildStep(), sql.OrderByField(field, opts...))
	}
}

// ByUserField orders the results by user field.
func ByUserField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newUserStep(), sql.OrderByField(field, opts...))
	}
}
func newGuildStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(GuildInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, GuildTable, GuildColumn),
	)
}
func newUserStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(UserInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, UserTable, UserColumn),
	)
}
