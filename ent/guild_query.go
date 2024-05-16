// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"database/sql/driver"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/sabafly/gobot/ent/chinchirosession"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/member"
	"github.com/sabafly/gobot/ent/messagepin"
	"github.com/sabafly/gobot/ent/messageremind"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepaneledit"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
	"github.com/sabafly/gobot/ent/user"
)

// GuildQuery is the builder for querying Guild entities.
type GuildQuery struct {
	config
	ctx                     *QueryContext
	order                   []guild.OrderOption
	inters                  []Interceptor
	predicates              []predicate.Guild
	withOwner               *UserQuery
	withMembers             *MemberQuery
	withMessagePins         *MessagePinQuery
	withReminds             *MessageRemindQuery
	withRolePanels          *RolePanelQuery
	withRolePanelPlacements *RolePanelPlacedQuery
	withRolePanelEdits      *RolePanelEditQuery
	withChinchiroSessions   *ChinchiroSessionQuery
	withFKs                 bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the GuildQuery builder.
func (gq *GuildQuery) Where(ps ...predicate.Guild) *GuildQuery {
	gq.predicates = append(gq.predicates, ps...)
	return gq
}

// Limit the number of records to be returned by this query.
func (gq *GuildQuery) Limit(limit int) *GuildQuery {
	gq.ctx.Limit = &limit
	return gq
}

// Offset to start from.
func (gq *GuildQuery) Offset(offset int) *GuildQuery {
	gq.ctx.Offset = &offset
	return gq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (gq *GuildQuery) Unique(unique bool) *GuildQuery {
	gq.ctx.Unique = &unique
	return gq
}

// Order specifies how the records should be ordered.
func (gq *GuildQuery) Order(o ...guild.OrderOption) *GuildQuery {
	gq.order = append(gq.order, o...)
	return gq
}

// QueryOwner chains the current query on the "owner" edge.
func (gq *GuildQuery) QueryOwner() *UserQuery {
	query := (&UserClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(user.Table, user.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, guild.OwnerTable, guild.OwnerColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryMembers chains the current query on the "members" edge.
func (gq *GuildQuery) QueryMembers() *MemberQuery {
	query := (&MemberClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(member.Table, member.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, guild.MembersTable, guild.MembersColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryMessagePins chains the current query on the "message_pins" edge.
func (gq *GuildQuery) QueryMessagePins() *MessagePinQuery {
	query := (&MessagePinClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(messagepin.Table, messagepin.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, guild.MessagePinsTable, guild.MessagePinsColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryReminds chains the current query on the "reminds" edge.
func (gq *GuildQuery) QueryReminds() *MessageRemindQuery {
	query := (&MessageRemindClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(messageremind.Table, messageremind.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, guild.RemindsTable, guild.RemindsColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryRolePanels chains the current query on the "role_panels" edge.
func (gq *GuildQuery) QueryRolePanels() *RolePanelQuery {
	query := (&RolePanelClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(rolepanel.Table, rolepanel.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, guild.RolePanelsTable, guild.RolePanelsColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryRolePanelPlacements chains the current query on the "role_panel_placements" edge.
func (gq *GuildQuery) QueryRolePanelPlacements() *RolePanelPlacedQuery {
	query := (&RolePanelPlacedClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(rolepanelplaced.Table, rolepanelplaced.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, guild.RolePanelPlacementsTable, guild.RolePanelPlacementsColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryRolePanelEdits chains the current query on the "role_panel_edits" edge.
func (gq *GuildQuery) QueryRolePanelEdits() *RolePanelEditQuery {
	query := (&RolePanelEditClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(rolepaneledit.Table, rolepaneledit.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, guild.RolePanelEditsTable, guild.RolePanelEditsColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryChinchiroSessions chains the current query on the "chinchiro_sessions" edge.
func (gq *GuildQuery) QueryChinchiroSessions() *ChinchiroSessionQuery {
	query := (&ChinchiroSessionClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(guild.Table, guild.FieldID, selector),
			sqlgraph.To(chinchirosession.Table, chinchirosession.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, guild.ChinchiroSessionsTable, guild.ChinchiroSessionsColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Guild entity from the query.
// Returns a *NotFoundError when no Guild was found.
func (gq *GuildQuery) First(ctx context.Context) (*Guild, error) {
	nodes, err := gq.Limit(1).All(setContextOp(ctx, gq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{guild.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (gq *GuildQuery) FirstX(ctx context.Context) *Guild {
	node, err := gq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Guild ID from the query.
// Returns a *NotFoundError when no Guild ID was found.
func (gq *GuildQuery) FirstID(ctx context.Context) (id snowflake.ID, err error) {
	var ids []snowflake.ID
	if ids, err = gq.Limit(1).IDs(setContextOp(ctx, gq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{guild.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (gq *GuildQuery) FirstIDX(ctx context.Context) snowflake.ID {
	id, err := gq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Guild entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Guild entity is found.
// Returns a *NotFoundError when no Guild entities are found.
func (gq *GuildQuery) Only(ctx context.Context) (*Guild, error) {
	nodes, err := gq.Limit(2).All(setContextOp(ctx, gq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{guild.Label}
	default:
		return nil, &NotSingularError{guild.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (gq *GuildQuery) OnlyX(ctx context.Context) *Guild {
	node, err := gq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Guild ID in the query.
// Returns a *NotSingularError when more than one Guild ID is found.
// Returns a *NotFoundError when no entities are found.
func (gq *GuildQuery) OnlyID(ctx context.Context) (id snowflake.ID, err error) {
	var ids []snowflake.ID
	if ids, err = gq.Limit(2).IDs(setContextOp(ctx, gq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{guild.Label}
	default:
		err = &NotSingularError{guild.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (gq *GuildQuery) OnlyIDX(ctx context.Context) snowflake.ID {
	id, err := gq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Guilds.
func (gq *GuildQuery) All(ctx context.Context) ([]*Guild, error) {
	ctx = setContextOp(ctx, gq.ctx, "All")
	if err := gq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*Guild, *GuildQuery]()
	return withInterceptors[[]*Guild](ctx, gq, qr, gq.inters)
}

// AllX is like All, but panics if an error occurs.
func (gq *GuildQuery) AllX(ctx context.Context) []*Guild {
	nodes, err := gq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Guild IDs.
func (gq *GuildQuery) IDs(ctx context.Context) (ids []snowflake.ID, err error) {
	if gq.ctx.Unique == nil && gq.path != nil {
		gq.Unique(true)
	}
	ctx = setContextOp(ctx, gq.ctx, "IDs")
	if err = gq.Select(guild.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (gq *GuildQuery) IDsX(ctx context.Context) []snowflake.ID {
	ids, err := gq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (gq *GuildQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, gq.ctx, "Count")
	if err := gq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, gq, querierCount[*GuildQuery](), gq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (gq *GuildQuery) CountX(ctx context.Context) int {
	count, err := gq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (gq *GuildQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, gq.ctx, "Exist")
	switch _, err := gq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (gq *GuildQuery) ExistX(ctx context.Context) bool {
	exist, err := gq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the GuildQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (gq *GuildQuery) Clone() *GuildQuery {
	if gq == nil {
		return nil
	}
	return &GuildQuery{
		config:                  gq.config,
		ctx:                     gq.ctx.Clone(),
		order:                   append([]guild.OrderOption{}, gq.order...),
		inters:                  append([]Interceptor{}, gq.inters...),
		predicates:              append([]predicate.Guild{}, gq.predicates...),
		withOwner:               gq.withOwner.Clone(),
		withMembers:             gq.withMembers.Clone(),
		withMessagePins:         gq.withMessagePins.Clone(),
		withReminds:             gq.withReminds.Clone(),
		withRolePanels:          gq.withRolePanels.Clone(),
		withRolePanelPlacements: gq.withRolePanelPlacements.Clone(),
		withRolePanelEdits:      gq.withRolePanelEdits.Clone(),
		withChinchiroSessions:   gq.withChinchiroSessions.Clone(),
		// clone intermediate query.
		sql:  gq.sql.Clone(),
		path: gq.path,
	}
}

// WithOwner tells the query-builder to eager-load the nodes that are connected to
// the "owner" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithOwner(opts ...func(*UserQuery)) *GuildQuery {
	query := (&UserClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withOwner = query
	return gq
}

// WithMembers tells the query-builder to eager-load the nodes that are connected to
// the "members" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithMembers(opts ...func(*MemberQuery)) *GuildQuery {
	query := (&MemberClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withMembers = query
	return gq
}

// WithMessagePins tells the query-builder to eager-load the nodes that are connected to
// the "message_pins" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithMessagePins(opts ...func(*MessagePinQuery)) *GuildQuery {
	query := (&MessagePinClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withMessagePins = query
	return gq
}

// WithReminds tells the query-builder to eager-load the nodes that are connected to
// the "reminds" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithReminds(opts ...func(*MessageRemindQuery)) *GuildQuery {
	query := (&MessageRemindClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withReminds = query
	return gq
}

// WithRolePanels tells the query-builder to eager-load the nodes that are connected to
// the "role_panels" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithRolePanels(opts ...func(*RolePanelQuery)) *GuildQuery {
	query := (&RolePanelClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withRolePanels = query
	return gq
}

// WithRolePanelPlacements tells the query-builder to eager-load the nodes that are connected to
// the "role_panel_placements" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithRolePanelPlacements(opts ...func(*RolePanelPlacedQuery)) *GuildQuery {
	query := (&RolePanelPlacedClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withRolePanelPlacements = query
	return gq
}

// WithRolePanelEdits tells the query-builder to eager-load the nodes that are connected to
// the "role_panel_edits" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithRolePanelEdits(opts ...func(*RolePanelEditQuery)) *GuildQuery {
	query := (&RolePanelEditClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withRolePanelEdits = query
	return gq
}

// WithChinchiroSessions tells the query-builder to eager-load the nodes that are connected to
// the "chinchiro_sessions" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GuildQuery) WithChinchiroSessions(opts ...func(*ChinchiroSessionQuery)) *GuildQuery {
	query := (&ChinchiroSessionClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withChinchiroSessions = query
	return gq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.Guild.Query().
//		GroupBy(guild.FieldName).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (gq *GuildQuery) GroupBy(field string, fields ...string) *GuildGroupBy {
	gq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &GuildGroupBy{build: gq}
	grbuild.flds = &gq.ctx.Fields
	grbuild.label = guild.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Name string `json:"name,omitempty"`
//	}
//
//	client.Guild.Query().
//		Select(guild.FieldName).
//		Scan(ctx, &v)
func (gq *GuildQuery) Select(fields ...string) *GuildSelect {
	gq.ctx.Fields = append(gq.ctx.Fields, fields...)
	sbuild := &GuildSelect{GuildQuery: gq}
	sbuild.label = guild.Label
	sbuild.flds, sbuild.scan = &gq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a GuildSelect configured with the given aggregations.
func (gq *GuildQuery) Aggregate(fns ...AggregateFunc) *GuildSelect {
	return gq.Select().Aggregate(fns...)
}

func (gq *GuildQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range gq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, gq); err != nil {
				return err
			}
		}
	}
	for _, f := range gq.ctx.Fields {
		if !guild.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if gq.path != nil {
		prev, err := gq.path(ctx)
		if err != nil {
			return err
		}
		gq.sql = prev
	}
	return nil
}

func (gq *GuildQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Guild, error) {
	var (
		nodes       = []*Guild{}
		withFKs     = gq.withFKs
		_spec       = gq.querySpec()
		loadedTypes = [8]bool{
			gq.withOwner != nil,
			gq.withMembers != nil,
			gq.withMessagePins != nil,
			gq.withReminds != nil,
			gq.withRolePanels != nil,
			gq.withRolePanelPlacements != nil,
			gq.withRolePanelEdits != nil,
			gq.withChinchiroSessions != nil,
		}
	)
	if gq.withOwner != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, guild.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*Guild).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &Guild{config: gq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, gq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := gq.withOwner; query != nil {
		if err := gq.loadOwner(ctx, query, nodes, nil,
			func(n *Guild, e *User) { n.Edges.Owner = e }); err != nil {
			return nil, err
		}
	}
	if query := gq.withMembers; query != nil {
		if err := gq.loadMembers(ctx, query, nodes,
			func(n *Guild) { n.Edges.Members = []*Member{} },
			func(n *Guild, e *Member) { n.Edges.Members = append(n.Edges.Members, e) }); err != nil {
			return nil, err
		}
	}
	if query := gq.withMessagePins; query != nil {
		if err := gq.loadMessagePins(ctx, query, nodes,
			func(n *Guild) { n.Edges.MessagePins = []*MessagePin{} },
			func(n *Guild, e *MessagePin) { n.Edges.MessagePins = append(n.Edges.MessagePins, e) }); err != nil {
			return nil, err
		}
	}
	if query := gq.withReminds; query != nil {
		if err := gq.loadReminds(ctx, query, nodes,
			func(n *Guild) { n.Edges.Reminds = []*MessageRemind{} },
			func(n *Guild, e *MessageRemind) { n.Edges.Reminds = append(n.Edges.Reminds, e) }); err != nil {
			return nil, err
		}
	}
	if query := gq.withRolePanels; query != nil {
		if err := gq.loadRolePanels(ctx, query, nodes,
			func(n *Guild) { n.Edges.RolePanels = []*RolePanel{} },
			func(n *Guild, e *RolePanel) { n.Edges.RolePanels = append(n.Edges.RolePanels, e) }); err != nil {
			return nil, err
		}
	}
	if query := gq.withRolePanelPlacements; query != nil {
		if err := gq.loadRolePanelPlacements(ctx, query, nodes,
			func(n *Guild) { n.Edges.RolePanelPlacements = []*RolePanelPlaced{} },
			func(n *Guild, e *RolePanelPlaced) {
				n.Edges.RolePanelPlacements = append(n.Edges.RolePanelPlacements, e)
			}); err != nil {
			return nil, err
		}
	}
	if query := gq.withRolePanelEdits; query != nil {
		if err := gq.loadRolePanelEdits(ctx, query, nodes,
			func(n *Guild) { n.Edges.RolePanelEdits = []*RolePanelEdit{} },
			func(n *Guild, e *RolePanelEdit) { n.Edges.RolePanelEdits = append(n.Edges.RolePanelEdits, e) }); err != nil {
			return nil, err
		}
	}
	if query := gq.withChinchiroSessions; query != nil {
		if err := gq.loadChinchiroSessions(ctx, query, nodes,
			func(n *Guild) { n.Edges.ChinchiroSessions = []*ChinchiroSession{} },
			func(n *Guild, e *ChinchiroSession) { n.Edges.ChinchiroSessions = append(n.Edges.ChinchiroSessions, e) }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (gq *GuildQuery) loadOwner(ctx context.Context, query *UserQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *User)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*Guild)
	for i := range nodes {
		if nodes[i].user_own_guilds == nil {
			continue
		}
		fk := *nodes[i].user_own_guilds
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(user.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "user_own_guilds" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (gq *GuildQuery) loadMembers(ctx context.Context, query *MemberQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *Member)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[snowflake.ID]*Guild)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.Member(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(guild.MembersColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.guild_members
		if fk == nil {
			return fmt.Errorf(`foreign-key "guild_members" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "guild_members" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (gq *GuildQuery) loadMessagePins(ctx context.Context, query *MessagePinQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *MessagePin)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[snowflake.ID]*Guild)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.MessagePin(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(guild.MessagePinsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.guild_message_pins
		if fk == nil {
			return fmt.Errorf(`foreign-key "guild_message_pins" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "guild_message_pins" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (gq *GuildQuery) loadReminds(ctx context.Context, query *MessageRemindQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *MessageRemind)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[snowflake.ID]*Guild)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.MessageRemind(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(guild.RemindsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.guild_reminds
		if fk == nil {
			return fmt.Errorf(`foreign-key "guild_reminds" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "guild_reminds" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (gq *GuildQuery) loadRolePanels(ctx context.Context, query *RolePanelQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *RolePanel)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[snowflake.ID]*Guild)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.RolePanel(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(guild.RolePanelsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.guild_role_panels
		if fk == nil {
			return fmt.Errorf(`foreign-key "guild_role_panels" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "guild_role_panels" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (gq *GuildQuery) loadRolePanelPlacements(ctx context.Context, query *RolePanelPlacedQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *RolePanelPlaced)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[snowflake.ID]*Guild)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.RolePanelPlaced(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(guild.RolePanelPlacementsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.guild_role_panel_placements
		if fk == nil {
			return fmt.Errorf(`foreign-key "guild_role_panel_placements" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "guild_role_panel_placements" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (gq *GuildQuery) loadRolePanelEdits(ctx context.Context, query *RolePanelEditQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *RolePanelEdit)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[snowflake.ID]*Guild)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.RolePanelEdit(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(guild.RolePanelEditsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.guild_role_panel_edits
		if fk == nil {
			return fmt.Errorf(`foreign-key "guild_role_panel_edits" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "guild_role_panel_edits" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (gq *GuildQuery) loadChinchiroSessions(ctx context.Context, query *ChinchiroSessionQuery, nodes []*Guild, init func(*Guild), assign func(*Guild, *ChinchiroSession)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[snowflake.ID]*Guild)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.ChinchiroSession(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(guild.ChinchiroSessionsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.guild_chinchiro_sessions
		if fk == nil {
			return fmt.Errorf(`foreign-key "guild_chinchiro_sessions" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "guild_chinchiro_sessions" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}

func (gq *GuildQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := gq.querySpec()
	_spec.Node.Columns = gq.ctx.Fields
	if len(gq.ctx.Fields) > 0 {
		_spec.Unique = gq.ctx.Unique != nil && *gq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, gq.driver, _spec)
}

func (gq *GuildQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(guild.Table, guild.Columns, sqlgraph.NewFieldSpec(guild.FieldID, field.TypeUint64))
	_spec.From = gq.sql
	if unique := gq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if gq.path != nil {
		_spec.Unique = true
	}
	if fields := gq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, guild.FieldID)
		for i := range fields {
			if fields[i] != guild.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := gq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := gq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := gq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := gq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (gq *GuildQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(gq.driver.Dialect())
	t1 := builder.Table(guild.Table)
	columns := gq.ctx.Fields
	if len(columns) == 0 {
		columns = guild.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if gq.sql != nil {
		selector = gq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if gq.ctx.Unique != nil && *gq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range gq.predicates {
		p(selector)
	}
	for _, p := range gq.order {
		p(selector)
	}
	if offset := gq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := gq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// GuildGroupBy is the group-by builder for Guild entities.
type GuildGroupBy struct {
	selector
	build *GuildQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (ggb *GuildGroupBy) Aggregate(fns ...AggregateFunc) *GuildGroupBy {
	ggb.fns = append(ggb.fns, fns...)
	return ggb
}

// Scan applies the selector query and scans the result into the given value.
func (ggb *GuildGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ggb.build.ctx, "GroupBy")
	if err := ggb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*GuildQuery, *GuildGroupBy](ctx, ggb.build, ggb, ggb.build.inters, v)
}

func (ggb *GuildGroupBy) sqlScan(ctx context.Context, root *GuildQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(ggb.fns))
	for _, fn := range ggb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*ggb.flds)+len(ggb.fns))
		for _, f := range *ggb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*ggb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ggb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// GuildSelect is the builder for selecting fields of Guild entities.
type GuildSelect struct {
	*GuildQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (gs *GuildSelect) Aggregate(fns ...AggregateFunc) *GuildSelect {
	gs.fns = append(gs.fns, fns...)
	return gs
}

// Scan applies the selector query and scans the result into the given value.
func (gs *GuildSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, gs.ctx, "Select")
	if err := gs.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*GuildQuery, *GuildSelect](ctx, gs.GuildQuery, gs, gs.inters, v)
}

func (gs *GuildSelect) sqlScan(ctx context.Context, root *GuildQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(gs.fns))
	for _, fn := range gs.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*gs.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := gs.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
