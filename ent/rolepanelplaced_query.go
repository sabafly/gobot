// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	snowflake "github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/predicate"
	"github.com/sabafly/gobot/ent/rolepanel"
	"github.com/sabafly/gobot/ent/rolepanelplaced"
)

// RolePanelPlacedQuery is the builder for querying RolePanelPlaced entities.
type RolePanelPlacedQuery struct {
	config
	ctx           *QueryContext
	order         []rolepanelplaced.OrderOption
	inters        []Interceptor
	predicates    []predicate.RolePanelPlaced
	withGuild     *GuildQuery
	withRolePanel *RolePanelQuery
	withFKs       bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the RolePanelPlacedQuery builder.
func (rppq *RolePanelPlacedQuery) Where(ps ...predicate.RolePanelPlaced) *RolePanelPlacedQuery {
	rppq.predicates = append(rppq.predicates, ps...)
	return rppq
}

// Limit the number of records to be returned by this query.
func (rppq *RolePanelPlacedQuery) Limit(limit int) *RolePanelPlacedQuery {
	rppq.ctx.Limit = &limit
	return rppq
}

// Offset to start from.
func (rppq *RolePanelPlacedQuery) Offset(offset int) *RolePanelPlacedQuery {
	rppq.ctx.Offset = &offset
	return rppq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (rppq *RolePanelPlacedQuery) Unique(unique bool) *RolePanelPlacedQuery {
	rppq.ctx.Unique = &unique
	return rppq
}

// Order specifies how the records should be ordered.
func (rppq *RolePanelPlacedQuery) Order(o ...rolepanelplaced.OrderOption) *RolePanelPlacedQuery {
	rppq.order = append(rppq.order, o...)
	return rppq
}

// QueryGuild chains the current query on the "guild" edge.
func (rppq *RolePanelPlacedQuery) QueryGuild() *GuildQuery {
	query := (&GuildClient{config: rppq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := rppq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := rppq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(rolepanelplaced.Table, rolepanelplaced.FieldID, selector),
			sqlgraph.To(guild.Table, guild.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, rolepanelplaced.GuildTable, rolepanelplaced.GuildColumn),
		)
		fromU = sqlgraph.SetNeighbors(rppq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryRolePanel chains the current query on the "role_panel" edge.
func (rppq *RolePanelPlacedQuery) QueryRolePanel() *RolePanelQuery {
	query := (&RolePanelClient{config: rppq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := rppq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := rppq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(rolepanelplaced.Table, rolepanelplaced.FieldID, selector),
			sqlgraph.To(rolepanel.Table, rolepanel.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, rolepanelplaced.RolePanelTable, rolepanelplaced.RolePanelColumn),
		)
		fromU = sqlgraph.SetNeighbors(rppq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first RolePanelPlaced entity from the query.
// Returns a *NotFoundError when no RolePanelPlaced was found.
func (rppq *RolePanelPlacedQuery) First(ctx context.Context) (*RolePanelPlaced, error) {
	nodes, err := rppq.Limit(1).All(setContextOp(ctx, rppq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{rolepanelplaced.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) FirstX(ctx context.Context) *RolePanelPlaced {
	node, err := rppq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first RolePanelPlaced ID from the query.
// Returns a *NotFoundError when no RolePanelPlaced ID was found.
func (rppq *RolePanelPlacedQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = rppq.Limit(1).IDs(setContextOp(ctx, rppq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{rolepanelplaced.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := rppq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single RolePanelPlaced entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one RolePanelPlaced entity is found.
// Returns a *NotFoundError when no RolePanelPlaced entities are found.
func (rppq *RolePanelPlacedQuery) Only(ctx context.Context) (*RolePanelPlaced, error) {
	nodes, err := rppq.Limit(2).All(setContextOp(ctx, rppq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{rolepanelplaced.Label}
	default:
		return nil, &NotSingularError{rolepanelplaced.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) OnlyX(ctx context.Context) *RolePanelPlaced {
	node, err := rppq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only RolePanelPlaced ID in the query.
// Returns a *NotSingularError when more than one RolePanelPlaced ID is found.
// Returns a *NotFoundError when no entities are found.
func (rppq *RolePanelPlacedQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = rppq.Limit(2).IDs(setContextOp(ctx, rppq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{rolepanelplaced.Label}
	default:
		err = &NotSingularError{rolepanelplaced.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := rppq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of RolePanelPlaceds.
func (rppq *RolePanelPlacedQuery) All(ctx context.Context) ([]*RolePanelPlaced, error) {
	ctx = setContextOp(ctx, rppq.ctx, "All")
	if err := rppq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*RolePanelPlaced, *RolePanelPlacedQuery]()
	return withInterceptors[[]*RolePanelPlaced](ctx, rppq, qr, rppq.inters)
}

// AllX is like All, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) AllX(ctx context.Context) []*RolePanelPlaced {
	nodes, err := rppq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of RolePanelPlaced IDs.
func (rppq *RolePanelPlacedQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if rppq.ctx.Unique == nil && rppq.path != nil {
		rppq.Unique(true)
	}
	ctx = setContextOp(ctx, rppq.ctx, "IDs")
	if err = rppq.Select(rolepanelplaced.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := rppq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (rppq *RolePanelPlacedQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, rppq.ctx, "Count")
	if err := rppq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, rppq, querierCount[*RolePanelPlacedQuery](), rppq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) CountX(ctx context.Context) int {
	count, err := rppq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (rppq *RolePanelPlacedQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, rppq.ctx, "Exist")
	switch _, err := rppq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (rppq *RolePanelPlacedQuery) ExistX(ctx context.Context) bool {
	exist, err := rppq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the RolePanelPlacedQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (rppq *RolePanelPlacedQuery) Clone() *RolePanelPlacedQuery {
	if rppq == nil {
		return nil
	}
	return &RolePanelPlacedQuery{
		config:        rppq.config,
		ctx:           rppq.ctx.Clone(),
		order:         append([]rolepanelplaced.OrderOption{}, rppq.order...),
		inters:        append([]Interceptor{}, rppq.inters...),
		predicates:    append([]predicate.RolePanelPlaced{}, rppq.predicates...),
		withGuild:     rppq.withGuild.Clone(),
		withRolePanel: rppq.withRolePanel.Clone(),
		// clone intermediate query.
		sql:  rppq.sql.Clone(),
		path: rppq.path,
	}
}

// WithGuild tells the query-builder to eager-load the nodes that are connected to
// the "guild" edge. The optional arguments are used to configure the query builder of the edge.
func (rppq *RolePanelPlacedQuery) WithGuild(opts ...func(*GuildQuery)) *RolePanelPlacedQuery {
	query := (&GuildClient{config: rppq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	rppq.withGuild = query
	return rppq
}

// WithRolePanel tells the query-builder to eager-load the nodes that are connected to
// the "role_panel" edge. The optional arguments are used to configure the query builder of the edge.
func (rppq *RolePanelPlacedQuery) WithRolePanel(opts ...func(*RolePanelQuery)) *RolePanelPlacedQuery {
	query := (&RolePanelClient{config: rppq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	rppq.withRolePanel = query
	return rppq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		MessageID snowflake.ID `json:"message_id,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.RolePanelPlaced.Query().
//		GroupBy(rolepanelplaced.FieldMessageID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (rppq *RolePanelPlacedQuery) GroupBy(field string, fields ...string) *RolePanelPlacedGroupBy {
	rppq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &RolePanelPlacedGroupBy{build: rppq}
	grbuild.flds = &rppq.ctx.Fields
	grbuild.label = rolepanelplaced.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		MessageID snowflake.ID `json:"message_id,omitempty"`
//	}
//
//	client.RolePanelPlaced.Query().
//		Select(rolepanelplaced.FieldMessageID).
//		Scan(ctx, &v)
func (rppq *RolePanelPlacedQuery) Select(fields ...string) *RolePanelPlacedSelect {
	rppq.ctx.Fields = append(rppq.ctx.Fields, fields...)
	sbuild := &RolePanelPlacedSelect{RolePanelPlacedQuery: rppq}
	sbuild.label = rolepanelplaced.Label
	sbuild.flds, sbuild.scan = &rppq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a RolePanelPlacedSelect configured with the given aggregations.
func (rppq *RolePanelPlacedQuery) Aggregate(fns ...AggregateFunc) *RolePanelPlacedSelect {
	return rppq.Select().Aggregate(fns...)
}

func (rppq *RolePanelPlacedQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range rppq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, rppq); err != nil {
				return err
			}
		}
	}
	for _, f := range rppq.ctx.Fields {
		if !rolepanelplaced.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if rppq.path != nil {
		prev, err := rppq.path(ctx)
		if err != nil {
			return err
		}
		rppq.sql = prev
	}
	return nil
}

func (rppq *RolePanelPlacedQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*RolePanelPlaced, error) {
	var (
		nodes       = []*RolePanelPlaced{}
		withFKs     = rppq.withFKs
		_spec       = rppq.querySpec()
		loadedTypes = [2]bool{
			rppq.withGuild != nil,
			rppq.withRolePanel != nil,
		}
	)
	if rppq.withGuild != nil || rppq.withRolePanel != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, rolepanelplaced.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*RolePanelPlaced).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &RolePanelPlaced{config: rppq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, rppq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := rppq.withGuild; query != nil {
		if err := rppq.loadGuild(ctx, query, nodes, nil,
			func(n *RolePanelPlaced, e *Guild) { n.Edges.Guild = e }); err != nil {
			return nil, err
		}
	}
	if query := rppq.withRolePanel; query != nil {
		if err := rppq.loadRolePanel(ctx, query, nodes, nil,
			func(n *RolePanelPlaced, e *RolePanel) { n.Edges.RolePanel = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (rppq *RolePanelPlacedQuery) loadGuild(ctx context.Context, query *GuildQuery, nodes []*RolePanelPlaced, init func(*RolePanelPlaced), assign func(*RolePanelPlaced, *Guild)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*RolePanelPlaced)
	for i := range nodes {
		if nodes[i].guild_role_panel_placements == nil {
			continue
		}
		fk := *nodes[i].guild_role_panel_placements
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(guild.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "guild_role_panel_placements" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (rppq *RolePanelPlacedQuery) loadRolePanel(ctx context.Context, query *RolePanelQuery, nodes []*RolePanelPlaced, init func(*RolePanelPlaced), assign func(*RolePanelPlaced, *RolePanel)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*RolePanelPlaced)
	for i := range nodes {
		if nodes[i].role_panel_placements == nil {
			continue
		}
		fk := *nodes[i].role_panel_placements
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(rolepanel.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "role_panel_placements" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (rppq *RolePanelPlacedQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := rppq.querySpec()
	_spec.Node.Columns = rppq.ctx.Fields
	if len(rppq.ctx.Fields) > 0 {
		_spec.Unique = rppq.ctx.Unique != nil && *rppq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, rppq.driver, _spec)
}

func (rppq *RolePanelPlacedQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(rolepanelplaced.Table, rolepanelplaced.Columns, sqlgraph.NewFieldSpec(rolepanelplaced.FieldID, field.TypeUUID))
	_spec.From = rppq.sql
	if unique := rppq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if rppq.path != nil {
		_spec.Unique = true
	}
	if fields := rppq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, rolepanelplaced.FieldID)
		for i := range fields {
			if fields[i] != rolepanelplaced.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := rppq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := rppq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := rppq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := rppq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (rppq *RolePanelPlacedQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(rppq.driver.Dialect())
	t1 := builder.Table(rolepanelplaced.Table)
	columns := rppq.ctx.Fields
	if len(columns) == 0 {
		columns = rolepanelplaced.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if rppq.sql != nil {
		selector = rppq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if rppq.ctx.Unique != nil && *rppq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range rppq.predicates {
		p(selector)
	}
	for _, p := range rppq.order {
		p(selector)
	}
	if offset := rppq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := rppq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// RolePanelPlacedGroupBy is the group-by builder for RolePanelPlaced entities.
type RolePanelPlacedGroupBy struct {
	selector
	build *RolePanelPlacedQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (rppgb *RolePanelPlacedGroupBy) Aggregate(fns ...AggregateFunc) *RolePanelPlacedGroupBy {
	rppgb.fns = append(rppgb.fns, fns...)
	return rppgb
}

// Scan applies the selector query and scans the result into the given value.
func (rppgb *RolePanelPlacedGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, rppgb.build.ctx, "GroupBy")
	if err := rppgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*RolePanelPlacedQuery, *RolePanelPlacedGroupBy](ctx, rppgb.build, rppgb, rppgb.build.inters, v)
}

func (rppgb *RolePanelPlacedGroupBy) sqlScan(ctx context.Context, root *RolePanelPlacedQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(rppgb.fns))
	for _, fn := range rppgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*rppgb.flds)+len(rppgb.fns))
		for _, f := range *rppgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*rppgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := rppgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// RolePanelPlacedSelect is the builder for selecting fields of RolePanelPlaced entities.
type RolePanelPlacedSelect struct {
	*RolePanelPlacedQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (rpps *RolePanelPlacedSelect) Aggregate(fns ...AggregateFunc) *RolePanelPlacedSelect {
	rpps.fns = append(rpps.fns, fns...)
	return rpps
}

// Scan applies the selector query and scans the result into the given value.
func (rpps *RolePanelPlacedSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, rpps.ctx, "Select")
	if err := rpps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*RolePanelPlacedQuery, *RolePanelPlacedSelect](ctx, rpps.RolePanelPlacedQuery, rpps, rpps.inters, v)
}

func (rpps *RolePanelPlacedSelect) sqlScan(ctx context.Context, root *RolePanelPlacedQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(rpps.fns))
	for _, fn := range rpps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*rpps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := rpps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}