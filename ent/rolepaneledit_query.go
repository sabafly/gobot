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
	"github.com/sabafly/gobot/ent/rolepaneledit"
)

// RolePanelEditQuery is the builder for querying RolePanelEdit entities.
type RolePanelEditQuery struct {
	config
	ctx        *QueryContext
	order      []rolepaneledit.OrderOption
	inters     []Interceptor
	predicates []predicate.RolePanelEdit
	withGuild  *GuildQuery
	withParent *RolePanelQuery
	withFKs    bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the RolePanelEditQuery builder.
func (rpeq *RolePanelEditQuery) Where(ps ...predicate.RolePanelEdit) *RolePanelEditQuery {
	rpeq.predicates = append(rpeq.predicates, ps...)
	return rpeq
}

// Limit the number of records to be returned by this query.
func (rpeq *RolePanelEditQuery) Limit(limit int) *RolePanelEditQuery {
	rpeq.ctx.Limit = &limit
	return rpeq
}

// Offset to start from.
func (rpeq *RolePanelEditQuery) Offset(offset int) *RolePanelEditQuery {
	rpeq.ctx.Offset = &offset
	return rpeq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (rpeq *RolePanelEditQuery) Unique(unique bool) *RolePanelEditQuery {
	rpeq.ctx.Unique = &unique
	return rpeq
}

// Order specifies how the records should be ordered.
func (rpeq *RolePanelEditQuery) Order(o ...rolepaneledit.OrderOption) *RolePanelEditQuery {
	rpeq.order = append(rpeq.order, o...)
	return rpeq
}

// QueryGuild chains the current query on the "guild" edge.
func (rpeq *RolePanelEditQuery) QueryGuild() *GuildQuery {
	query := (&GuildClient{config: rpeq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := rpeq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := rpeq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(rolepaneledit.Table, rolepaneledit.FieldID, selector),
			sqlgraph.To(guild.Table, guild.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, rolepaneledit.GuildTable, rolepaneledit.GuildColumn),
		)
		fromU = sqlgraph.SetNeighbors(rpeq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryParent chains the current query on the "parent" edge.
func (rpeq *RolePanelEditQuery) QueryParent() *RolePanelQuery {
	query := (&RolePanelClient{config: rpeq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := rpeq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := rpeq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(rolepaneledit.Table, rolepaneledit.FieldID, selector),
			sqlgraph.To(rolepanel.Table, rolepanel.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, rolepaneledit.ParentTable, rolepaneledit.ParentColumn),
		)
		fromU = sqlgraph.SetNeighbors(rpeq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first RolePanelEdit entity from the query.
// Returns a *NotFoundError when no RolePanelEdit was found.
func (rpeq *RolePanelEditQuery) First(ctx context.Context) (*RolePanelEdit, error) {
	nodes, err := rpeq.Limit(1).All(setContextOp(ctx, rpeq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{rolepaneledit.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) FirstX(ctx context.Context) *RolePanelEdit {
	node, err := rpeq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first RolePanelEdit ID from the query.
// Returns a *NotFoundError when no RolePanelEdit ID was found.
func (rpeq *RolePanelEditQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = rpeq.Limit(1).IDs(setContextOp(ctx, rpeq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{rolepaneledit.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := rpeq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single RolePanelEdit entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one RolePanelEdit entity is found.
// Returns a *NotFoundError when no RolePanelEdit entities are found.
func (rpeq *RolePanelEditQuery) Only(ctx context.Context) (*RolePanelEdit, error) {
	nodes, err := rpeq.Limit(2).All(setContextOp(ctx, rpeq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{rolepaneledit.Label}
	default:
		return nil, &NotSingularError{rolepaneledit.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) OnlyX(ctx context.Context) *RolePanelEdit {
	node, err := rpeq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only RolePanelEdit ID in the query.
// Returns a *NotSingularError when more than one RolePanelEdit ID is found.
// Returns a *NotFoundError when no entities are found.
func (rpeq *RolePanelEditQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = rpeq.Limit(2).IDs(setContextOp(ctx, rpeq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{rolepaneledit.Label}
	default:
		err = &NotSingularError{rolepaneledit.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := rpeq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of RolePanelEdits.
func (rpeq *RolePanelEditQuery) All(ctx context.Context) ([]*RolePanelEdit, error) {
	ctx = setContextOp(ctx, rpeq.ctx, "All")
	if err := rpeq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*RolePanelEdit, *RolePanelEditQuery]()
	return withInterceptors[[]*RolePanelEdit](ctx, rpeq, qr, rpeq.inters)
}

// AllX is like All, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) AllX(ctx context.Context) []*RolePanelEdit {
	nodes, err := rpeq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of RolePanelEdit IDs.
func (rpeq *RolePanelEditQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if rpeq.ctx.Unique == nil && rpeq.path != nil {
		rpeq.Unique(true)
	}
	ctx = setContextOp(ctx, rpeq.ctx, "IDs")
	if err = rpeq.Select(rolepaneledit.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := rpeq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (rpeq *RolePanelEditQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, rpeq.ctx, "Count")
	if err := rpeq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, rpeq, querierCount[*RolePanelEditQuery](), rpeq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) CountX(ctx context.Context) int {
	count, err := rpeq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (rpeq *RolePanelEditQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, rpeq.ctx, "Exist")
	switch _, err := rpeq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (rpeq *RolePanelEditQuery) ExistX(ctx context.Context) bool {
	exist, err := rpeq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the RolePanelEditQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (rpeq *RolePanelEditQuery) Clone() *RolePanelEditQuery {
	if rpeq == nil {
		return nil
	}
	return &RolePanelEditQuery{
		config:     rpeq.config,
		ctx:        rpeq.ctx.Clone(),
		order:      append([]rolepaneledit.OrderOption{}, rpeq.order...),
		inters:     append([]Interceptor{}, rpeq.inters...),
		predicates: append([]predicate.RolePanelEdit{}, rpeq.predicates...),
		withGuild:  rpeq.withGuild.Clone(),
		withParent: rpeq.withParent.Clone(),
		// clone intermediate query.
		sql:  rpeq.sql.Clone(),
		path: rpeq.path,
	}
}

// WithGuild tells the query-builder to eager-load the nodes that are connected to
// the "guild" edge. The optional arguments are used to configure the query builder of the edge.
func (rpeq *RolePanelEditQuery) WithGuild(opts ...func(*GuildQuery)) *RolePanelEditQuery {
	query := (&GuildClient{config: rpeq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	rpeq.withGuild = query
	return rpeq
}

// WithParent tells the query-builder to eager-load the nodes that are connected to
// the "parent" edge. The optional arguments are used to configure the query builder of the edge.
func (rpeq *RolePanelEditQuery) WithParent(opts ...func(*RolePanelQuery)) *RolePanelEditQuery {
	query := (&RolePanelClient{config: rpeq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	rpeq.withParent = query
	return rpeq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		ChannelID snowflake.ID `json:"channel_id,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.RolePanelEdit.Query().
//		GroupBy(rolepaneledit.FieldChannelID).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (rpeq *RolePanelEditQuery) GroupBy(field string, fields ...string) *RolePanelEditGroupBy {
	rpeq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &RolePanelEditGroupBy{build: rpeq}
	grbuild.flds = &rpeq.ctx.Fields
	grbuild.label = rolepaneledit.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		ChannelID snowflake.ID `json:"channel_id,omitempty"`
//	}
//
//	client.RolePanelEdit.Query().
//		Select(rolepaneledit.FieldChannelID).
//		Scan(ctx, &v)
func (rpeq *RolePanelEditQuery) Select(fields ...string) *RolePanelEditSelect {
	rpeq.ctx.Fields = append(rpeq.ctx.Fields, fields...)
	sbuild := &RolePanelEditSelect{RolePanelEditQuery: rpeq}
	sbuild.label = rolepaneledit.Label
	sbuild.flds, sbuild.scan = &rpeq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a RolePanelEditSelect configured with the given aggregations.
func (rpeq *RolePanelEditQuery) Aggregate(fns ...AggregateFunc) *RolePanelEditSelect {
	return rpeq.Select().Aggregate(fns...)
}

func (rpeq *RolePanelEditQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range rpeq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, rpeq); err != nil {
				return err
			}
		}
	}
	for _, f := range rpeq.ctx.Fields {
		if !rolepaneledit.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if rpeq.path != nil {
		prev, err := rpeq.path(ctx)
		if err != nil {
			return err
		}
		rpeq.sql = prev
	}
	return nil
}

func (rpeq *RolePanelEditQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*RolePanelEdit, error) {
	var (
		nodes       = []*RolePanelEdit{}
		withFKs     = rpeq.withFKs
		_spec       = rpeq.querySpec()
		loadedTypes = [2]bool{
			rpeq.withGuild != nil,
			rpeq.withParent != nil,
		}
	)
	if rpeq.withGuild != nil || rpeq.withParent != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, rolepaneledit.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*RolePanelEdit).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &RolePanelEdit{config: rpeq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, rpeq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := rpeq.withGuild; query != nil {
		if err := rpeq.loadGuild(ctx, query, nodes, nil,
			func(n *RolePanelEdit, e *Guild) { n.Edges.Guild = e }); err != nil {
			return nil, err
		}
	}
	if query := rpeq.withParent; query != nil {
		if err := rpeq.loadParent(ctx, query, nodes, nil,
			func(n *RolePanelEdit, e *RolePanel) { n.Edges.Parent = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (rpeq *RolePanelEditQuery) loadGuild(ctx context.Context, query *GuildQuery, nodes []*RolePanelEdit, init func(*RolePanelEdit), assign func(*RolePanelEdit, *Guild)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*RolePanelEdit)
	for i := range nodes {
		if nodes[i].guild_role_panel_edits == nil {
			continue
		}
		fk := *nodes[i].guild_role_panel_edits
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
			return fmt.Errorf(`unexpected foreign-key "guild_role_panel_edits" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (rpeq *RolePanelEditQuery) loadParent(ctx context.Context, query *RolePanelQuery, nodes []*RolePanelEdit, init func(*RolePanelEdit), assign func(*RolePanelEdit, *RolePanel)) error {
	ids := make([]uuid.UUID, 0, len(nodes))
	nodeids := make(map[uuid.UUID][]*RolePanelEdit)
	for i := range nodes {
		if nodes[i].role_panel_edit == nil {
			continue
		}
		fk := *nodes[i].role_panel_edit
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
			return fmt.Errorf(`unexpected foreign-key "role_panel_edit" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (rpeq *RolePanelEditQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := rpeq.querySpec()
	_spec.Node.Columns = rpeq.ctx.Fields
	if len(rpeq.ctx.Fields) > 0 {
		_spec.Unique = rpeq.ctx.Unique != nil && *rpeq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, rpeq.driver, _spec)
}

func (rpeq *RolePanelEditQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(rolepaneledit.Table, rolepaneledit.Columns, sqlgraph.NewFieldSpec(rolepaneledit.FieldID, field.TypeUUID))
	_spec.From = rpeq.sql
	if unique := rpeq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if rpeq.path != nil {
		_spec.Unique = true
	}
	if fields := rpeq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, rolepaneledit.FieldID)
		for i := range fields {
			if fields[i] != rolepaneledit.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := rpeq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := rpeq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := rpeq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := rpeq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (rpeq *RolePanelEditQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(rpeq.driver.Dialect())
	t1 := builder.Table(rolepaneledit.Table)
	columns := rpeq.ctx.Fields
	if len(columns) == 0 {
		columns = rolepaneledit.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if rpeq.sql != nil {
		selector = rpeq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if rpeq.ctx.Unique != nil && *rpeq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range rpeq.predicates {
		p(selector)
	}
	for _, p := range rpeq.order {
		p(selector)
	}
	if offset := rpeq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := rpeq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// RolePanelEditGroupBy is the group-by builder for RolePanelEdit entities.
type RolePanelEditGroupBy struct {
	selector
	build *RolePanelEditQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (rpegb *RolePanelEditGroupBy) Aggregate(fns ...AggregateFunc) *RolePanelEditGroupBy {
	rpegb.fns = append(rpegb.fns, fns...)
	return rpegb
}

// Scan applies the selector query and scans the result into the given value.
func (rpegb *RolePanelEditGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, rpegb.build.ctx, "GroupBy")
	if err := rpegb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*RolePanelEditQuery, *RolePanelEditGroupBy](ctx, rpegb.build, rpegb, rpegb.build.inters, v)
}

func (rpegb *RolePanelEditGroupBy) sqlScan(ctx context.Context, root *RolePanelEditQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(rpegb.fns))
	for _, fn := range rpegb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*rpegb.flds)+len(rpegb.fns))
		for _, f := range *rpegb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*rpegb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := rpegb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// RolePanelEditSelect is the builder for selecting fields of RolePanelEdit entities.
type RolePanelEditSelect struct {
	*RolePanelEditQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (rpes *RolePanelEditSelect) Aggregate(fns ...AggregateFunc) *RolePanelEditSelect {
	rpes.fns = append(rpes.fns, fns...)
	return rpes
}

// Scan applies the selector query and scans the result into the given value.
func (rpes *RolePanelEditSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, rpes.ctx, "Select")
	if err := rpes.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*RolePanelEditQuery, *RolePanelEditSelect](ctx, rpes.RolePanelEditQuery, rpes, rpes.inters, v)
}

func (rpes *RolePanelEditSelect) sqlScan(ctx context.Context, root *RolePanelEditQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(rpes.fns))
	for _, fn := range rpes.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*rpes.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := rpes.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
