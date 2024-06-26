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
	"github.com/google/uuid"
	"github.com/sabafly/gobot/ent/chinchiroplayer"
	"github.com/sabafly/gobot/ent/chinchirosession"
	"github.com/sabafly/gobot/ent/guild"
	"github.com/sabafly/gobot/ent/predicate"
)

// ChinchiroSessionQuery is the builder for querying ChinchiroSession entities.
type ChinchiroSessionQuery struct {
	config
	ctx         *QueryContext
	order       []chinchirosession.OrderOption
	inters      []Interceptor
	predicates  []predicate.ChinchiroSession
	withGuild   *GuildQuery
	withPlayers *ChinchiroPlayerQuery
	withFKs     bool
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the ChinchiroSessionQuery builder.
func (csq *ChinchiroSessionQuery) Where(ps ...predicate.ChinchiroSession) *ChinchiroSessionQuery {
	csq.predicates = append(csq.predicates, ps...)
	return csq
}

// Limit the number of records to be returned by this query.
func (csq *ChinchiroSessionQuery) Limit(limit int) *ChinchiroSessionQuery {
	csq.ctx.Limit = &limit
	return csq
}

// Offset to start from.
func (csq *ChinchiroSessionQuery) Offset(offset int) *ChinchiroSessionQuery {
	csq.ctx.Offset = &offset
	return csq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (csq *ChinchiroSessionQuery) Unique(unique bool) *ChinchiroSessionQuery {
	csq.ctx.Unique = &unique
	return csq
}

// Order specifies how the records should be ordered.
func (csq *ChinchiroSessionQuery) Order(o ...chinchirosession.OrderOption) *ChinchiroSessionQuery {
	csq.order = append(csq.order, o...)
	return csq
}

// QueryGuild chains the current query on the "guild" edge.
func (csq *ChinchiroSessionQuery) QueryGuild() *GuildQuery {
	query := (&GuildClient{config: csq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := csq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := csq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(chinchirosession.Table, chinchirosession.FieldID, selector),
			sqlgraph.To(guild.Table, guild.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, chinchirosession.GuildTable, chinchirosession.GuildColumn),
		)
		fromU = sqlgraph.SetNeighbors(csq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryPlayers chains the current query on the "players" edge.
func (csq *ChinchiroSessionQuery) QueryPlayers() *ChinchiroPlayerQuery {
	query := (&ChinchiroPlayerClient{config: csq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := csq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := csq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(chinchirosession.Table, chinchirosession.FieldID, selector),
			sqlgraph.To(chinchiroplayer.Table, chinchiroplayer.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, chinchirosession.PlayersTable, chinchirosession.PlayersColumn),
		)
		fromU = sqlgraph.SetNeighbors(csq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first ChinchiroSession entity from the query.
// Returns a *NotFoundError when no ChinchiroSession was found.
func (csq *ChinchiroSessionQuery) First(ctx context.Context) (*ChinchiroSession, error) {
	nodes, err := csq.Limit(1).All(setContextOp(ctx, csq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{chinchirosession.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) FirstX(ctx context.Context) *ChinchiroSession {
	node, err := csq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first ChinchiroSession ID from the query.
// Returns a *NotFoundError when no ChinchiroSession ID was found.
func (csq *ChinchiroSessionQuery) FirstID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = csq.Limit(1).IDs(setContextOp(ctx, csq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{chinchirosession.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) FirstIDX(ctx context.Context) uuid.UUID {
	id, err := csq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single ChinchiroSession entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one ChinchiroSession entity is found.
// Returns a *NotFoundError when no ChinchiroSession entities are found.
func (csq *ChinchiroSessionQuery) Only(ctx context.Context) (*ChinchiroSession, error) {
	nodes, err := csq.Limit(2).All(setContextOp(ctx, csq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{chinchirosession.Label}
	default:
		return nil, &NotSingularError{chinchirosession.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) OnlyX(ctx context.Context) *ChinchiroSession {
	node, err := csq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only ChinchiroSession ID in the query.
// Returns a *NotSingularError when more than one ChinchiroSession ID is found.
// Returns a *NotFoundError when no entities are found.
func (csq *ChinchiroSessionQuery) OnlyID(ctx context.Context) (id uuid.UUID, err error) {
	var ids []uuid.UUID
	if ids, err = csq.Limit(2).IDs(setContextOp(ctx, csq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{chinchirosession.Label}
	default:
		err = &NotSingularError{chinchirosession.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) OnlyIDX(ctx context.Context) uuid.UUID {
	id, err := csq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of ChinchiroSessions.
func (csq *ChinchiroSessionQuery) All(ctx context.Context) ([]*ChinchiroSession, error) {
	ctx = setContextOp(ctx, csq.ctx, "All")
	if err := csq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*ChinchiroSession, *ChinchiroSessionQuery]()
	return withInterceptors[[]*ChinchiroSession](ctx, csq, qr, csq.inters)
}

// AllX is like All, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) AllX(ctx context.Context) []*ChinchiroSession {
	nodes, err := csq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of ChinchiroSession IDs.
func (csq *ChinchiroSessionQuery) IDs(ctx context.Context) (ids []uuid.UUID, err error) {
	if csq.ctx.Unique == nil && csq.path != nil {
		csq.Unique(true)
	}
	ctx = setContextOp(ctx, csq.ctx, "IDs")
	if err = csq.Select(chinchirosession.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) IDsX(ctx context.Context) []uuid.UUID {
	ids, err := csq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (csq *ChinchiroSessionQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, csq.ctx, "Count")
	if err := csq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, csq, querierCount[*ChinchiroSessionQuery](), csq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) CountX(ctx context.Context) int {
	count, err := csq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (csq *ChinchiroSessionQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, csq.ctx, "Exist")
	switch _, err := csq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("ent: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (csq *ChinchiroSessionQuery) ExistX(ctx context.Context) bool {
	exist, err := csq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the ChinchiroSessionQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (csq *ChinchiroSessionQuery) Clone() *ChinchiroSessionQuery {
	if csq == nil {
		return nil
	}
	return &ChinchiroSessionQuery{
		config:      csq.config,
		ctx:         csq.ctx.Clone(),
		order:       append([]chinchirosession.OrderOption{}, csq.order...),
		inters:      append([]Interceptor{}, csq.inters...),
		predicates:  append([]predicate.ChinchiroSession{}, csq.predicates...),
		withGuild:   csq.withGuild.Clone(),
		withPlayers: csq.withPlayers.Clone(),
		// clone intermediate query.
		sql:  csq.sql.Clone(),
		path: csq.path,
	}
}

// WithGuild tells the query-builder to eager-load the nodes that are connected to
// the "guild" edge. The optional arguments are used to configure the query builder of the edge.
func (csq *ChinchiroSessionQuery) WithGuild(opts ...func(*GuildQuery)) *ChinchiroSessionQuery {
	query := (&GuildClient{config: csq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	csq.withGuild = query
	return csq
}

// WithPlayers tells the query-builder to eager-load the nodes that are connected to
// the "players" edge. The optional arguments are used to configure the query builder of the edge.
func (csq *ChinchiroSessionQuery) WithPlayers(opts ...func(*ChinchiroPlayerQuery)) *ChinchiroSessionQuery {
	query := (&ChinchiroPlayerClient{config: csq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	csq.withPlayers = query
	return csq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Turn int `json:"turn,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.ChinchiroSession.Query().
//		GroupBy(chinchirosession.FieldTurn).
//		Aggregate(ent.Count()).
//		Scan(ctx, &v)
func (csq *ChinchiroSessionQuery) GroupBy(field string, fields ...string) *ChinchiroSessionGroupBy {
	csq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &ChinchiroSessionGroupBy{build: csq}
	grbuild.flds = &csq.ctx.Fields
	grbuild.label = chinchirosession.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Turn int `json:"turn,omitempty"`
//	}
//
//	client.ChinchiroSession.Query().
//		Select(chinchirosession.FieldTurn).
//		Scan(ctx, &v)
func (csq *ChinchiroSessionQuery) Select(fields ...string) *ChinchiroSessionSelect {
	csq.ctx.Fields = append(csq.ctx.Fields, fields...)
	sbuild := &ChinchiroSessionSelect{ChinchiroSessionQuery: csq}
	sbuild.label = chinchirosession.Label
	sbuild.flds, sbuild.scan = &csq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a ChinchiroSessionSelect configured with the given aggregations.
func (csq *ChinchiroSessionQuery) Aggregate(fns ...AggregateFunc) *ChinchiroSessionSelect {
	return csq.Select().Aggregate(fns...)
}

func (csq *ChinchiroSessionQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range csq.inters {
		if inter == nil {
			return fmt.Errorf("ent: uninitialized interceptor (forgotten import ent/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, csq); err != nil {
				return err
			}
		}
	}
	for _, f := range csq.ctx.Fields {
		if !chinchirosession.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
		}
	}
	if csq.path != nil {
		prev, err := csq.path(ctx)
		if err != nil {
			return err
		}
		csq.sql = prev
	}
	return nil
}

func (csq *ChinchiroSessionQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*ChinchiroSession, error) {
	var (
		nodes       = []*ChinchiroSession{}
		withFKs     = csq.withFKs
		_spec       = csq.querySpec()
		loadedTypes = [2]bool{
			csq.withGuild != nil,
			csq.withPlayers != nil,
		}
	)
	if csq.withGuild != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, chinchirosession.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*ChinchiroSession).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &ChinchiroSession{config: csq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, csq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := csq.withGuild; query != nil {
		if err := csq.loadGuild(ctx, query, nodes, nil,
			func(n *ChinchiroSession, e *Guild) { n.Edges.Guild = e }); err != nil {
			return nil, err
		}
	}
	if query := csq.withPlayers; query != nil {
		if err := csq.loadPlayers(ctx, query, nodes,
			func(n *ChinchiroSession) { n.Edges.Players = []*ChinchiroPlayer{} },
			func(n *ChinchiroSession, e *ChinchiroPlayer) { n.Edges.Players = append(n.Edges.Players, e) }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (csq *ChinchiroSessionQuery) loadGuild(ctx context.Context, query *GuildQuery, nodes []*ChinchiroSession, init func(*ChinchiroSession), assign func(*ChinchiroSession, *Guild)) error {
	ids := make([]snowflake.ID, 0, len(nodes))
	nodeids := make(map[snowflake.ID][]*ChinchiroSession)
	for i := range nodes {
		if nodes[i].guild_chinchiro_sessions == nil {
			continue
		}
		fk := *nodes[i].guild_chinchiro_sessions
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
			return fmt.Errorf(`unexpected foreign-key "guild_chinchiro_sessions" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (csq *ChinchiroSessionQuery) loadPlayers(ctx context.Context, query *ChinchiroPlayerQuery, nodes []*ChinchiroSession, init func(*ChinchiroSession), assign func(*ChinchiroSession, *ChinchiroPlayer)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[uuid.UUID]*ChinchiroSession)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	query.Where(predicate.ChinchiroPlayer(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(chinchirosession.PlayersColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.chinchiro_session_players
		if fk == nil {
			return fmt.Errorf(`foreign-key "chinchiro_session_players" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "chinchiro_session_players" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}

func (csq *ChinchiroSessionQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := csq.querySpec()
	_spec.Node.Columns = csq.ctx.Fields
	if len(csq.ctx.Fields) > 0 {
		_spec.Unique = csq.ctx.Unique != nil && *csq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, csq.driver, _spec)
}

func (csq *ChinchiroSessionQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(chinchirosession.Table, chinchirosession.Columns, sqlgraph.NewFieldSpec(chinchirosession.FieldID, field.TypeUUID))
	_spec.From = csq.sql
	if unique := csq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if csq.path != nil {
		_spec.Unique = true
	}
	if fields := csq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, chinchirosession.FieldID)
		for i := range fields {
			if fields[i] != chinchirosession.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := csq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := csq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := csq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := csq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (csq *ChinchiroSessionQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(csq.driver.Dialect())
	t1 := builder.Table(chinchirosession.Table)
	columns := csq.ctx.Fields
	if len(columns) == 0 {
		columns = chinchirosession.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if csq.sql != nil {
		selector = csq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if csq.ctx.Unique != nil && *csq.ctx.Unique {
		selector.Distinct()
	}
	for _, p := range csq.predicates {
		p(selector)
	}
	for _, p := range csq.order {
		p(selector)
	}
	if offset := csq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := csq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ChinchiroSessionGroupBy is the group-by builder for ChinchiroSession entities.
type ChinchiroSessionGroupBy struct {
	selector
	build *ChinchiroSessionQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (csgb *ChinchiroSessionGroupBy) Aggregate(fns ...AggregateFunc) *ChinchiroSessionGroupBy {
	csgb.fns = append(csgb.fns, fns...)
	return csgb
}

// Scan applies the selector query and scans the result into the given value.
func (csgb *ChinchiroSessionGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, csgb.build.ctx, "GroupBy")
	if err := csgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*ChinchiroSessionQuery, *ChinchiroSessionGroupBy](ctx, csgb.build, csgb, csgb.build.inters, v)
}

func (csgb *ChinchiroSessionGroupBy) sqlScan(ctx context.Context, root *ChinchiroSessionQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(csgb.fns))
	for _, fn := range csgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*csgb.flds)+len(csgb.fns))
		for _, f := range *csgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*csgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := csgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// ChinchiroSessionSelect is the builder for selecting fields of ChinchiroSession entities.
type ChinchiroSessionSelect struct {
	*ChinchiroSessionQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (css *ChinchiroSessionSelect) Aggregate(fns ...AggregateFunc) *ChinchiroSessionSelect {
	css.fns = append(css.fns, fns...)
	return css
}

// Scan applies the selector query and scans the result into the given value.
func (css *ChinchiroSessionSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, css.ctx, "Select")
	if err := css.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*ChinchiroSessionQuery, *ChinchiroSessionSelect](ctx, css.ChinchiroSessionQuery, css, css.inters, v)
}

func (css *ChinchiroSessionSelect) sqlScan(ctx context.Context, root *ChinchiroSessionQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(css.fns))
	for _, fn := range css.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*css.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := css.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
