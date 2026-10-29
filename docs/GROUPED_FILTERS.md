# Opt-in grouped collection filters

Existing applications need no changes. `CollectionParams`, `CollectionFilter`,
`CollectionFilterField`, `CollectionSorter`, `PgFilters`, `PgFilter`, `PgSelect`,
and all existing interfaces retain their layouts/signatures. Positional Go
literals and third-party builders still compile. Existing parsing functions,
operator fallback, unknown-field handling and flat SQL generation are untouched.

Use the additive `CollectionParamsWithWhere` request and `PgGroupedSelect`
builder only on endpoints that should accept explicit grouping. No cursor token,
endpoint, scrolling behavior, response format or implicit sorting is added.

## Minimal opt-in for modelbind

Change the request argument to `modelbind.CollectionParamsWithWhere`, construct
`pg.NewPgGroupedSelect` instead of `pg.NewPgSelect`, and call
`modelbind.BindCollectionSelectModelWithWhere` instead of
`modelbind.BindCollectionSelectModel`. Build through the wrapper's `BuildSQL` or
`BuildCollectionSQL`; do not unwrap its embedded `PgSelect` to generate SQL.

```go
var params modelbind.CollectionParamsWithWhere
if err := json.Unmarshal(requestJSON, &params); err != nil {
	return err
}
query := pg.NewPgGroupedSelect(
	&OrderList{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{},
)
if err := modelbind.BindCollectionSelectModelWithWhere(query, params); err != nil {
	return err
}
args := []any{}
dataSQL, aggregateSQL, err := query.BuildCollectionSQL(&args)
```

The model still supplies normal JSON field annotations, validation constraints,
and `CollectionAgg()`. Mark date-only time fields with `dateType:"date"`.
Normal sorter validation and limit behavior are reused. Builders are request-local.
Check binding/build errors before executing queries.

`Where` is `json.RawMessage`, retaining the distinction between an omitted
expression and explicit `null`, and permitting strict parsing of only the new
syntax. It is JSON data, never SQL. Ordinary `json.Unmarshal` into the extended
request is supported; expression validation occurs during binding. Call
`ValidateCollectionWhere` for an early structure/operator check without a model.
`ParseCollectionWhere(model, raw, table)` adds metadata/value validation and
returns an opaque `pg.FilterExpression` for lower-level `SetWhere` use.

When `where` is absent, the wrapper delegates directly to `PgSelect`'s existing
SQL builder. Existing requests also work through the opted-in endpoint.
The original request/parsing/binding APIs continue to ignore an unknown `where`
just as they ignore any other unknown property; updating dependencies alone does
not enable the new feature.

## Expressions

A leaf contains exactly `f`, `o` and `v`:

```json
{ "f": "customer_id", "o": "e", "v": 10 }
```

A group contains exactly one of `and` or `or`, with a non-empty array of child
expressions. Fields can repeat freely in different leaves. Group order determines
parameter order. The maximum depth is 64 edges below the root (a root leaf has
depth zero). Null expressions, empty/mixed groups, unknown or duplicate node
members, omitted leaf members, invalid fields/operators and excess depth fail
with errors containing paths such as `where.or[1].and[0]`.

Each field must pass both the existing SQL identifier/JSON-path sanitizer and
model metadata lookup; values use the same metadata validators as legacy
collection filters. Clients cannot provide raw SQL or field mappings. SQL values
are bound as parameters; identifiers are never taken from parameter values.
The server-only `table` argument to `ParseCollectionWhere` is validated too.

Supported operator codes are `e`, `l`, `g`, `le`, `ge`, `lk`, `ilk`, `ne`, `i`,
`in`, `incl`, `any`, `has`, `overlap`, `contains`, and `fts`. Each is compiled by
the existing collection parser and `PgFilter`, retaining its exact behavior,
value validation and parameter representation. No new implicit conversion is
introduced. In particular:

- `i` with `v: null` means `IS NULL`; `in` with `v: null` means `IS NOT NULL`.
- `e`/`ne` with null retain bound-null comparison semantics; they are not changed
  into null predicates. Use `i`/`in` for null tests.
- `incl` means `column = ANY(parameter_array)` in these versions.
- The existing compiler aliases the SQL operator for `any` and `has`, so both
  currently follow the `parameter = ANY(array_column)` branch. Grouped filters
  deliberately retain this behavior. Use `incl` for array-parameter membership.
- Full-text search retains `to_tsquery('russian', parameter)`.

The separate `access.Predicate` API has different null/membership semantics and
fewer collection operators. Rather than changing that API, grouped client
conditions reuse collection leaf compilation and the existing PostgreSQL scoped
query/policy and transactional parameter-building infrastructure.

## Existing request: unchanged

[legacy.json](../examples/grouped-filters/legacy.json) remains valid with either
old or new APIs:

```json
{
	"filter": [{ "j": "and", "f": { "customer_id": { "o": "e", "v": 10 } } }],
	"sorter": [{ "f": "id", "d": "d" }],
	"from": 0,
	"count": 100
}
```

Legacy objects remain flat. A sequence `A OR B AND C` retains SQL precedence
`A OR (B AND C)`; no object-level grouping is inferred, default joins are not
changed and legacy multi-field map iteration is not reordered.

## Compound date-and-ID cursor

The full request is in
[date-id-cursor.json](../examples/grouped-filters/date-id-cursor.json):

```json
{
	"where": {
		"and": [
			{ "f": "customer_id", "o": "e", "v": 10 },
			{
				"or": [
					{ "f": "order_date", "o": "l", "v": "2026-10-29" },
					{
						"and": [
							{ "f": "order_date", "o": "e", "v": "2026-10-29" },
							{ "f": "id", "o": "l", "v": 1500 }
						]
					}
				]
			}
		]
	},
	"sorter": [{ "f": "order_date", "d": "d" }, { "f": "id", "d": "d" }],
	"from": 0,
	"count": 100
}
```

This produces the equivalent of:

```sql
WHERE customer_id = $1
	AND (order_date < $2 OR (order_date = $3 AND id < $4))
ORDER BY order_date DESC, id DESC
LIMIT 100
```

Parameters are `[10, "2026-10-29", "2026-10-29", 1500]`. Applications construct
bounds from the last row and keep a unique final sorting key. Change comparisons
as appropriate for ascending/mixed sorting; nullable sort columns need explicit
null-ordering conditions. There is no automatic cursor calculation.

## Search filters plus cursor bounds

[search-and-cursor.json](../examples/grouped-filters/search-and-cursor.json)
keeps an existing flat search for `name ILIKE '%apple%' OR comment ILIKE '%apple%'`
in `filter`, and puts only the date-and-ID cursor expression in `where`.
Both search columns must exist in that application's model metadata.

The combination is always:

```sql
(complete legacy expression) AND (complete grouped expression)
```

Existing server request predicates and authorization policies remain independent
AND restrictions outside the client expressions. Neither a client OR nor
`SetWhere` can replace an installed policy. The policy may appear before the
new group in emitted SQL; AND composition gives the same scope.

Data and aggregate SQL share exactly one built WHERE expression and one argument
slice. Sort/offset/count apply only to data. Aggregates therefore reflect all
active filters, including cursor bounds, before the page limit.

## Verification

Run `go test ./...` and `go test -race ./...` with Go 1.25+. Tests cover old
positional literals/interfaces, legacy precedence and unknown members, nested
cursor conditions, repeated fields, all collection operators, nulls, parameters,
validation paths, depth limits, fail-closed policy errors and matching data/totals
scope. `ExampleBindCollectionSelectModelWithWhere` is an executable Go example.
The webapp repository additionally tests both HTTP query forms, query execution
through `ds.Conn`, policy composition, and optional real PostgreSQL execution.
