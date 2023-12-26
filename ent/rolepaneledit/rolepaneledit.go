// Code generated by ent, DO NOT EDIT.

package rolepaneledit

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the rolepaneledit type in the database.
	Label = "role_panel_edit"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldChannelID holds the string denoting the channel_id field in the database.
	FieldChannelID = "channel_id"
	// FieldEmojiAuthor holds the string denoting the emoji_author field in the database.
	FieldEmojiAuthor = "emoji_author"
	// FieldToken holds the string denoting the token field in the database.
	FieldToken = "token"
	// FieldSelectedRole holds the string denoting the selected_role field in the database.
	FieldSelectedRole = "selected_role"
	// FieldModified holds the string denoting the modified field in the database.
	FieldModified = "modified"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldRoles holds the string denoting the roles field in the database.
	FieldRoles = "roles"
	// EdgeGuild holds the string denoting the guild edge name in mutations.
	EdgeGuild = "guild"
	// EdgeParent holds the string denoting the parent edge name in mutations.
	EdgeParent = "parent"
	// Table holds the table name of the rolepaneledit in the database.
	Table = "role_panel_edits"
	// GuildTable is the table that holds the guild relation/edge.
	GuildTable = "role_panel_edits"
	// GuildInverseTable is the table name for the Guild entity.
	// It exists in this package in order to avoid circular dependency with the "guild" package.
	GuildInverseTable = "guilds"
	// GuildColumn is the table column denoting the guild relation/edge.
	GuildColumn = "guild_role_panel_edits"
	// ParentTable is the table that holds the parent relation/edge.
	ParentTable = "role_panel_edits"
	// ParentInverseTable is the table name for the RolePanel entity.
	// It exists in this package in order to avoid circular dependency with the "rolepanel" package.
	ParentInverseTable = "role_panels"
	// ParentColumn is the table column denoting the parent relation/edge.
	ParentColumn = "role_panel_edit"
)

// Columns holds all SQL columns for rolepaneledit fields.
var Columns = []string{
	FieldID,
	FieldChannelID,
	FieldEmojiAuthor,
	FieldToken,
	FieldSelectedRole,
	FieldModified,
	FieldName,
	FieldDescription,
	FieldRoles,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "role_panel_edits"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"guild_role_panel_edits",
	"role_panel_edit",
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
	// DefaultModified holds the default value on creation for the "modified" field.
	DefaultModified bool
	// NameValidator is a validator for the "name" field. It is called by the builders before save.
	NameValidator func(string) error
	// DescriptionValidator is a validator for the "description" field. It is called by the builders before save.
	DescriptionValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// OrderOption defines the ordering options for the RolePanelEdit queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByChannelID orders the results by the channel_id field.
func ByChannelID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldChannelID, opts...).ToFunc()
}

// ByEmojiAuthor orders the results by the emoji_author field.
func ByEmojiAuthor(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldEmojiAuthor, opts...).ToFunc()
}

// ByToken orders the results by the token field.
func ByToken(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldToken, opts...).ToFunc()
}

// BySelectedRole orders the results by the selected_role field.
func BySelectedRole(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSelectedRole, opts...).ToFunc()
}

// ByModified orders the results by the modified field.
func ByModified(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldModified, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByGuildField orders the results by guild field.
func ByGuildField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newGuildStep(), sql.OrderByField(field, opts...))
	}
}

// ByParentField orders the results by parent field.
func ByParentField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newParentStep(), sql.OrderByField(field, opts...))
	}
}
func newGuildStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(GuildInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, GuildTable, GuildColumn),
	)
}
func newParentStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ParentInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, ParentTable, ParentColumn),
	)
}
