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
	"github.com/sabafly/gobot/ent/user"
	"github.com/sabafly/gobot/ent/wordsuffix"
)

// WordSuffixQuery is the builder for querying WordSuffix entities.
type WordSuffixQuery struct {
	config
	ctx        *QueryContext
	order      []wordsuffix.OrderOption
	inters     []Interceptor
	predicates []predicate.WordSuffix
	withGuild  *GuildQuery
	withOwner  *UserQuery
	withFKs    bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the WordSuffixQuery builder.
func (wsq *WordSuffixQuery) Where(ps ...predicate.WordSuffix) *WordSuffixQuery {
	wsq.predicates = append(wsq.predicates, ps...)
	return wsq
}

// Limit the number of records to be returned by this query.
func (wsq *WordSuffixQuery) Limit(limit int) *WordSuffixQuery {
	wsq.ctx.Limit = &limit
	return wsq
}

// Offset to start from.
func (wsq *WordSuffixQuery) Offset(offset int) *WordSuffixQuery {
	wsq.ctx.Offset = &offset
	return wsq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (wsq *WordSuffixQuery) Unique(unique bool) *WordSuffixQuery {
	wsq.ctx.Unique = &unique
	return wsq
}

// Order specifies how the records should be ordered.
func (wsq *WordSuffixQuery) Order(o ...wordsuffix.OrderOption) *WordSuffixQuery {
	wsq.order = append(wsq.order, o...)
	return wsq
}

// QueryGuild chains the current query on the "guild" edge.
func (wsq *WordSuffixQuery) QueryGuild() *GuildQuery {
	query := (&GuildClient{config: wsq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := wsq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := wsq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(wordsuffix.Table, wordsuffix.FieldID, selector),
			sqlgraph.To(guild.Table, guild.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, wordsuffix.GuildTable, wordsuffix.GuildColumn),
		)
		fromU = sqlgraph.SetNeighbors(wsq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryOwner chains the current query on the "owner" edge.
func (wsq *WordSuffixQuery) QueryOwner() *UserQuery {
	query := (&UserClient{config: wsq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := wsq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := wsq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(wordsuffix.Table, wordsuffix.FieldID, selector),
			sqlgraph.To(user.Table, user.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, wordsuffix.OwnerTable, wordsuffix.OwnerColumn),
		)
		fromU = sqlgraph.SetNeighbors(wsq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first WordSuffix entity from the query.
// Returns a *NotFoundError when no WordSuffix was found.
func (wsq *WordSuffixQuery) First(ctx context.Context) (*WordSuffix, error) {
	nodes, err := wsq.Limit(1).All(setContextOp(ctx, wsq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{wordsuffix.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (wsq *WordSuffixQuery) FirstX(ctx context.Context) *WordSuffix {
	node, err := wsq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first WordSuffix ID from the query.
// Returns a *NotFoundError when no WordSuffix ID was found.
func (wsq *WordSuffixQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = wsq.Limit(1).IDs(setContextOp(ctx, wsq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{wordsuffix.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (wsq *WordSuffixQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := wsq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single WordSuffix entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one WordSuffix entity is found.
// Returns a *NotFoundError when no WordSuffix entities are found.
func (wsq *WordSuffixQuery) Only(ctx context.Context) (*WordSuffix, error) {
	nodes, err := wsq.Limit(2).All(setContextOp(ctx, wsq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{wordsuffix.Label}
	default:
		return nil, &NotSingularError{wordsuffix.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (wsq *WordSuffixQuery) OnlyX(ctx context.Context) *WordSuffix {
	node, err := wsq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only WordSuffix ID in the query.
// Returns a *NotSingularError when more than one WordSuffix ID is found.
// Returns a *NotFoundError when no entities are found.
func (wsq *WordSuffixQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = wsq.Limit(2).IDs(setContextOp(ctx, wsq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{wordsuffix.Label}
	default:
		err = &NotSingularError{wordsuffix.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (wsq *WordSuffixQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := wsq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of WordSuffixes.
func (wsq *WordSuffixQuery) All(ctx context.Context) ([]*WordSuffix, error) {
	ctx = setContextOp(ctx, wsq.ctx, "All")
	if err := wsq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*WordSuffix, *WordSuffixQuery]()
	return withInterceptors[[]*WordSuffix](ctx, wsq, qr, wsq.inters)
}

// AllX is like All, but panics if an error occurs.
func (wsq *WordSuffixQuery) AllX(ctx context.Context) []*WordSuffix {
	nodes, err := wsq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of WordSuffix IDs.
func (wsq *WordSuffixQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if wsq.ctx.Unique == nil && wsq.path != nil {
		wsq.Unique(true)
	}
	ctx = setContextOp(ctx, wsq.ctx, "IDs")
	if err = wsq.Select(wordsuffix.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (wsq *WordSuffixQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := wsq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (wsq *WordSuffixQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, wsq.ctx, "Count")
	if err := wsq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, wsq, querierCount[*WordSuffixQuery](), wsq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (wsq *WordSuffixQuery) CountX(ctx context.Context) int {
	count, err := wsq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (wsq *WordSuffixQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, wsq.ctx, "Exist")
	switch _, err := wsq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (wsq *WordSuffixQuery) ExistX(ctx context.Context) bool {
	exist, err := wsq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the WordSuffixQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (wsq *WordSuffixQuery) Clone() *WordSuffixQuery {
	if wsq == nil {
		return nil
	}
	return &WordSuffixQuery{
		config:     wsq.config,
		ctx:        wsq.ctx.Clone(),
		order:      append([]wordsuffix.OrderOption{}, wsq.order...),
		inters:     append([]Interceptor{}, wsq.inters...),
		predicates: append([]predicate.WordSuffix{}, wsq.predicates...),
		withGuild:  wsq.withGuild.Clone(),
		withOwner:  wsq.withOwner.Clone(),
		// clone intermediate query.
		sql:  wsq.sql.Clone(),
		path: wsq.path,
	}
}

// WithGuild tells the query-builder to eager-load the nodes that are connected to
// the "guild" edge. The optional arguments are used to configure the query builder of the edge.
func (wsq *WordSuffixQuery) WithGuild(opts ...func(*GuildQuery)) *WordSuffixQuery {
	query := (&GuildClient{config: wsq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	wsq.withGuild = query
	return wsq
}

// WithOwner tells the query-builder to eager-load the nodes that are connected to
// the "owner" edge. The optional arguments are used to configure the query builder of the edge.
func (wsq *WordSuffixQuery) WithOwner(opts ...func(*UserQuery)) *WordSuffixQuery {
	query := (&UserClient{config: wsq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	wsq.withOwner = query
	return wsq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Suffix string `json:"suffix,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.WordSuffix.Query().
//		GroupBy(wordsuffix.FieldSuffix).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (wsq *WordSuffixQuery) GroupBy(field string, fields ...string) *WordSuffixGroupBy {
	wsq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &WordSuffixGroupBy{build: wsq}
	grbuild.flds = &wsq.ctx.Fields
	grbuild.label = wordsuffix.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Suffix string `json:"suffix,omitempty"`
//	}
//
//	client.WordSuffix.Query().
//		Select(wordsuffix.FieldSuffix).
//		Scan(ctx, &v)
func (wsq *WordSuffixQuery) Select(fields ...string) *WordSuffixSelect {
	wsq.ctx.Fields = append(wsq.ctx.Fields, fields...)
	sbuild := &WordSuffixSelect{WordSuffixQuery: wsq}
	sbuild.label = wordsuffix.Label
	sbuild.flds, sbuild.scan = &wsq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a WordSuffixSelect configured with the given aggregations.
func (wsq *WordSuffixQuery) Aggregate(fns ...AggregateFunc) *WordSuffixSelect {
	return wsq.Select().Aggregate(fns...)
}

func (wsq *WordSuffixQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range wsq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, wsq); err != nil {
				return err
			}
		}
	}
	for _, f := range wsq.ctx.Fields {
		if !wordsuffix.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if wsq.path != nil {
		prev, err := wsq.path(ctx)
		if err != nil {
			return err
		}
		wsq.sql = prev
	}
	return nil
}

func (wsq *WordSuffixQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*WordSuffix, error) {
	var (
		nodes       = []*WordSuffix{}
		withFKs     = wsq.withFKs
		_spec       = wsq.querySpec()
		loadedTypes = [2]bool{
			wsq.withGuild != nil,
			wsq.withOwner != nil,
		}
	)
	if wsq.withOwner != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, wordsuffix.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*WordSuffix).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &WordSuffix{config: wsq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, wsq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := wsq.withGuild; query != nil {
		if err := wsq.loadGuild(ctx, query, nodes, nil,
			func(n *WordSuffix, e *Guild) { n.Edges.Guild = e }); err != nil {
			return nil, err
		}
	}
	if query := wsq.withOwner; query != nil {
		if err := wsq.loadOwner(ctx, query, nodes, nil,
			func(n *WordSuffix, e *User) { n.Edges.Owner = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (wsq *WordSuffixQuery) loadGuild(ctx context.Context, query *GuildQuery, nodes []*WordSuffix, init func(*WordSuffix), assign func(*WordSuffix, *Guild)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*WordSuffix)
	for i := range nodes {
		if nodes[i].GuildID == nil {
			continue
		}
		fk := *nodes[i].GuildID
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
			return fmt.Errorf(`unexpected foreign-key "guild_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (wsq *WordSuffixQuery) loadOwner(ctx context.Context, query *UserQuery, nodes []*WordSuffix, init func(*WordSuffix), assign func(*WordSuffix, *User)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*WordSuffix)
	for i := range nodes {
		if nodes[i].user_word_suffix == nil {
			continue
		}
		fk := *nodes[i].user_word_suffix
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
			return fmt.Errorf(`unexpected foreign-key "user_word_suffix" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (wsq *WordSuffixQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := wsq.querySpec()
	_spec.Node.Columns = wsq.ctx.Fields
	if len(wsq.ctx.Fields) > 0 {
		_spec.Unique = wsq.ctx.Unique != nil && *wsq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, wsq.driver, _spec)
}

func (wsq *WordSuffixQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(wordsuffix.Table, wordsuffix.Columns, sqlgraph.NewFieldSpec(wordsuffix.FieldID, field.TypeUUID))
	_spec.From = wsq.sql
	if unique := wsq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if wsq.path != nil {
		_spec.Unique = true
	}
	if fields := wsq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, wordsuffix.FieldID)
		for i := range fields {
			if fields[i] != wordsuffix.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if wsq.withGuild != nil {
			_spec.Node.AddColumnOnce(wordsuffix.FieldGuildID)
		}
	}
	if ps := wsq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := wsq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := wsq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := wsq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (wsq *WordSuffixQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(wsq.driver.Dialect())
	t1 := builder.Table(wordsuffix.Table)
	columns := wsq.ctx.Fields
	if len(columns) == 0 {
		columns = wordsuffix.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if wsq.sql != nil {
		selector = wsq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if wsq.ctx.Unique != nil && *wsq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range wsq.predicates {
		p(selector)
	}
	for _, p := range wsq.order {
		p(selector)
	}
	if offset := wsq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := wsq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// WordSuffixGroupBy is the group-by builder for WordSuffix entities.
type WordSuffixGroupBy struct {
	selector
	build *WordSuffixQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (wsgb *WordSuffixGroupBy) Aggregate(fns ...AggregateFunc) *WordSuffixGroupBy {
	wsgb.fns = append(wsgb.fns, fns...)
	return wsgb
}

// Scan applies the selector query and scans the result into the given value.
func (wsgb *WordSuffixGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, wsgb.build.ctx, "GroupBy")
	if err := wsgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*WordSuffixQuery, *WordSuffixGroupBy](ctx, wsgb.build, wsgb, wsgb.build.inters, v)
}

func (wsgb *WordSuffixGroupBy) sqlScan(ctx context.Context, root *WordSuffixQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(wsgb.fns))
	for _, fn := range wsgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*wsgb.flds)+len(wsgb.fns))
		for _, f := range *wsgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*wsgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := wsgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// WordSuffixSelect is the builder for selecting fields of WordSuffix entities.
type WordSuffixSelect struct {
	*WordSuffixQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (wss *WordSuffixSelect) Aggregate(fns ...AggregateFunc) *WordSuffixSelect {
	wss.fns = append(wss.fns, fns...)
	return wss
}

// Scan applies the selector query and scans the result into the given value.
func (wss *WordSuffixSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, wss.ctx, "Select")
	if err := wss.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*WordSuffixQuery, *WordSuffixSelect](ctx, wss.WordSuffixQuery, wss, wss.inters, v)
}

func (wss *WordSuffixSelect) sqlScan(ctx context.Context, root *WordSuffixQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(wss.fns))
	for _, fn := range wss.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*wss.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := wss.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}