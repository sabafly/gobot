// Code generated by ent, DO NOT EDIT.

package rolepanel

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the rolepanel type in the database.
	Label = "role_panel"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldRoles holds the string denoting the roles field in the database.
	FieldRoles = "roles"
	// EdgeGuild holds the string denoting the guild edge name in mutations.
	EdgeGuild = "guild"
	// EdgePlacements holds the string denoting the placements edge name in mutations.
	EdgePlacements = "placements"
	// EdgeEdit holds the string denoting the edit edge name in mutations.
	EdgeEdit = "edit"
	// Table holds the table name of the rolepanel in the database.
	Table = "role_panels"
	// GuildTable is the table that holds the guild relation/edge.
	GuildTable = "role_panels"
	// GuildInverseTable is the table name for the Guild entity.
	// It exists in this package in order to avoid circular dependency with the "guild" package.
	GuildInverseTable = "guilds"
	// GuildColumn is the table column denoting the guild relation/edge.
	GuildColumn = "guild_role_panels"
	// PlacementsTable is the table that holds the placements relation/edge.
	PlacementsTable = "role_panel_placeds"
	// PlacementsInverseTable is the table name for the RolePanelPlaced entity.
	// It exists in this package in order to avoid circular dependency with the "rolepanelplaced" package.
	PlacementsInverseTable = "role_panel_placeds"
	// PlacementsColumn is the table column denoting the placements relation/edge.
	PlacementsColumn = "role_panel_placements"
	// EditTable is the table that holds the edit relation/edge.
	EditTable = "role_panel_edits"
	// EditInverseTable is the table name for the RolePanelEdit entity.
	// It exists in this package in order to avoid circular dependency with the "rolepaneledit" package.
	EditInverseTable = "role_panel_edits"
	// EditColumn is the table column denoting the edit relation/edge.
	EditColumn = "role_panel_edit"
)

// Columns holds all SQL columns for rolepanel fields.
var Columns = []string{
	FieldID,
	FieldName,
	FieldDescription,
	FieldRoles,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "role_panels"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"guild_role_panels",
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
	// NameValidator is a validator for the "name" field. It is called by the builders before save.
	NameValidator func(string) error
	// DescriptionValidator is a validator for the "description" field. It is called by the builders before save.
	DescriptionValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// OrderOption defines the ordering options for the RolePanel queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
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

// ByPlacementsCount orders the results by placements count.
func ByPlacementsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newPlacementsStep(), opts...)
	}
}

// ByPlacements orders the results by placements terms.
func ByPlacements(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newPlacementsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByEditField orders the results by edit field.
func ByEditField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEditStep(), sql.OrderByField(field, opts...))
	}
}
func newGuildStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(GuildInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, GuildTable, GuildColumn),
	)
}
func newPlacementsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(PlacementsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, PlacementsTable, PlacementsColumn),
	)
}
func newEditStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EditInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, false, EditTable, EditColumn),
	)
}
