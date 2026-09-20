# Resolved access policies

`modelbind/access` and the PostgreSQL builders now support server-supplied row
predicates and scalar write rules. These APIs are additive and opt-in. Existing
constructors, CRUD interfaces, `AddField`, `SQL`, and `PgFilters` remain available.
Existing calls without an installed policy retain their legacy behavior.

This is the **modelbind layer**, not automatic user/role authorization. The
application (or the future `webapp` integration) must resolve a trusted principal,
select the operation's policy, and attach it to every protected builder.
`AccessResource()` alone does not enforce a policy. Arbitrary SQL and the separate
custom delete SQL in `webapp` are not intercepted by this implementation.

## Core APIs

| Package | API | Purpose |
| --- | --- | --- |
| `access` | `Policy`, `Operation`, `Decision` | Resolved operation-specific authorization. |
| `access` | `Eq`, `NotEq`, `Less`, `LessOrEqual`, `Greater`, `GreaterOrEqual`, `In`, `IsNull`, `IsNotNull`, `And`, `Or`, `True`, `False` | Closed predicate tree with no raw SQL node. |
| `access` | `WriteRules`, `Default`, `Enforce`, `Allowed`, `Immutable` | New input-value rules, separate from existing-row access. |
| `pg` | `SetAccessPolicy`, `PolicyBinding`, `FieldMap` | Install a policy and explicit column mappings. |
| `pg` | `SetRequestPredicate` | Add a grouped caller predicate without replacing the policy. |
| `pg` | `CompilePredicate` | Compile a parameterized expression for custom query composition. |
| `pg` | `BuildSQL`, `BuildCollectionSQL` | Error-returning alternatives to legacy rendering methods. |
| `pg` | `PgInsert.SetField`, `PgUpdate.SetField`, `PgAssigners.Set` | Replace an assignment in place and collapse duplicates. |
| `modelbind` | `PrepareModelInput` | Construct effective typed input without mutating submitted input. |
| `modelbind` | `BindInsertModelInputWithPolicy`, `BindUpdateModelInputWithPolicy` | Prepare and validate input using an already-installed write policy. |
| `modelbind` | `RequireModelKeys` | Validate caller keys independently of the row policy. |
| `types` | `AccessScopedModel`, `AccessWritePolicy` | Optional integration capabilities; existing CRUD interfaces are unchanged. |

## Read scope

```go
import (
	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/pg"
)

// customerID must come from authenticated, trusted server-side identity data.
policy := access.Scope(access.Eq("customer_id", customerID))

query := pg.NewPgSelect(orderList, filters, sorters, limit)
query.AddField("o.id", nil)
query.AddField("o.number", nil)
query.AddAggField("count(*)", nil)

// orderList.Relation() returns "orders o" in this example.
if err := query.SetAccessPolicy(policy, pg.PolicyBinding{
	Rows: pg.FieldMap{"customer_id": "o.customer_id"},
}); err != nil {
	return err
}

params := []any{}
dataSQL, totalSQL, err := query.BuildCollectionSQL(&params)
if err != nil {
	return err
}
```

Given client filters `status = 'draft' OR urgent = true`, the effective predicate is:

```sql
WHERE
	((o.status = $1) OR (o.urgent = $2))
	AND (o.customer_id = $3)
```

The same predicate and parameter positions are used for data and aggregate
queries. The aggregate query has no list pagination. A policy column does not
need to appear in the SELECT projection or input model.

Attach the read policy to `PgDetailSelect` as well. A protected list does not
implicitly protect a separate detail query.

For a projection, supply its own explicit column mapping. There is no attempt to
parse `Relation()` or infer a mapping from a Go type name. Logical predicate field
IDs are simple identifiers; aliases and supported JSON paths live in `FieldMap`.
All referenced logical fields, including an empty `In` set's field, must be mapped.

## Insert: prepare before validation

```go
package example

import (
	"github.com/dronm/modelbind"
	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/pg"
)

type OrderInput struct {
	ID         *int    `json:"id,omitempty" srvCalc:""`
	Name       *string `json:"name,omitempty" required:""`
	CustomerID *int    `json:"customer_id,omitempty" required:""`
}

func (*OrderInput) Relation() string {
	return "orders"
}

func BuildOrderInsert(
	input modelbind.ModelInput[*OrderInput],
	customerID int,
) (modelbind.ModelInput[*OrderInput], string, []any, error) {
	insert := pg.NewPgInsert(input.Model)
	policy := access.ForInsert(access.WriteRules{
		"customer_id": access.Enforce(customerID),
	})
	if err := insert.SetAccessPolicy(policy, pg.PolicyBinding{
		Writes: pg.FieldMap{"customer_id": "customer_id"},
	}); err != nil {
		return modelbind.ModelInput[*OrderInput]{}, "", nil, err
	}

	// Do not validate the original input first: customer_id can be absent.
	effective, err := modelbind.BindInsertModelInputWithPolicy(input, insert)
	if err != nil {
		return modelbind.ModelInput[*OrderInput]{}, "", nil, err
	}

	params := []any{}
	query, err := insert.BuildSQL(&params)
	return effective, query, params, err
}
```

`Enforce(customerID)` fills an absent insert value, accepts an equal submitted
value, and rejects a conflicting value or NULL (unless the enforced value itself
is NULL). It does not silently overwrite conflicts.

Use the returned `effective.Model` as the operation's result model. `RETURNING`
scan targets created by policy-aware binding point to this effective model;
`insert.Model()` still references the model supplied to the constructor. The
original request model and its presence tracking remain unchanged.

`PrepareModelInput` can be called directly for a separate preparation stage. It
only prepares data: the builder must still have its policy installed to enforce
rules again when SQL is generated.

The convenience binders require rule IDs to match input metadata IDs (normally
JSON tags), and normal field-to-column binding still follows those tags. For
custom remapping or a policy column absent from the input Go struct, use the
lower-level builder with an explicit `PolicyBinding.Writes` mapping. It can add
an enforced/default INSERT column without a matching model field:

```go
insert := pg.NewPgInsert(orderModel)
insert.AddField("name", "new order")
if err := insert.SetAccessPolicy(access.ForInsert(access.WriteRules{
	"owner": access.Enforce(customerID),
}), pg.PolicyBinding{
	Writes: pg.FieldMap{"owner": "customer_id"},
}); err != nil {
	return err
}
params := []any{}
query, err := insert.BuildSQL(&params)
```

In contrast, `PrepareModelInput` deliberately rejects unknown input fields,
rather than silently dropping their rules. It also rejects rules on `srvCalc`
fields: those are database-generated/RETURNING fields, not application-supplied
ownership fields. Direct builders do not inspect model metadata; their caller is
responsible for choosing writable columns.

## Update: existing rows and submitted values

```go
policy := access.Policy{
	Decision: access.Restricted,
	Rows: access.Eq("customer_id", customerID),
	Writes: access.WriteRules{
		"customer_id": access.Enforce(customerID),
	},
}

update := pg.NewPgUpdate(input.Model)
if err := update.SetAccessPolicy(policy, pg.PolicyBinding{
	Rows: pg.FieldMap{"customer_id": "customer_id"},
	Writes: pg.FieldMap{"customer_id": "customer_id"},
}); err != nil {
	return err
}

effective, err := modelbind.BindUpdateModelInputWithPolicy(key, input, update)
if err != nil {
	return err
}
params := []any{}
query, err := update.BuildSQL(&params)
```

The row predicate is added to the key predicate with AND. If `customer_id` is
absent from the PATCH, it is not added to SET. A submitted different customer or
NULL fails. Rules are rechecked on every SQL build, so a later `AddField` or
`SetField` cannot override them.

**Rows are not automatically new-row checks.** A policy with only
`Rows: Eq("customer_id", customerID)` does not prevent changing customer_id. Pair
ownership scopes with `Enforce`, `Allowed`, or `Immutable` as appropriate.

`RequireModelKeys` requires at least one annotated scalar key and rejects missing
or NULL members of composite keys. It does not assume that zero is an invalid
identifier. Protected update/delete builders independently require a caller
filter or nontrivial request predicate; the policy alone cannot supply that
safeguard. This builder-level check is not proof of a primary-key lookup. Bulk
operations must explicitly provide their caller target predicate.

## Write rule semantics

| Rule | INSERT absent | INSERT present | UPDATE absent | UPDATE present |
| --- | --- | --- | --- | --- |
| `Default(v)` | Supply v. | Keep value, including NULL. | Invalid rule for UPDATE. | Invalid rule for UPDATE. |
| `Enforce(v)` | Supply v. | Require equality. | Leave unchanged. | Require equality. |
| `Allowed(values...)` | Reject; no arbitrary default. | Require membership. | Leave unchanged. | Require membership. |
| `Immutable()` | Invalid rule for INSERT. | Invalid rule for INSERT. | Leave unchanged. | Reject, even when it might equal the stored value. |

Each field has one rule in this first implementation. Default is not an access
restriction. More complex rule composition should be explicit application logic,
not conflicting map entries that overwrite each other.

`Allowed()` with no members rejects every submitted value. NULL is allowed only
when explicitly included in a write-rule set. For a row `In` predicate, use
`Or(In(...), IsNull(...))` to include SQL NULL; NULL members in `In` are rejected.

Supported policy values are strings (including named enum aliases), booleans,
integers, finite floats, `time.Time`, scalar pointers, and explicit NULL. Integer
widths are compared exactly, including signed/unsigned boundaries. Strings are
not parsed as numbers, and floats are not coerced into integers. Typed input
preparation rejects overflow and lossy conversion. Maps, arrays/slices as scalar
parameters, custom database value wrappers, and raw SQL values are not supported.

Presence must come from decoding or a correctly populated `AbsentFieldSet`.
Without tracked presence, every model field is considered present; nil must not
be guessed to mean omission.

## Grouped deletes and custom predicates

```go
remove := pg.NewPgDelete(orderItemModel, nil)
keys := access.Or(
	access.And(access.Eq("document_id", 1), access.Eq("line_no", 2)),
	access.And(access.Eq("document_id", 3), access.Eq("line_no", 4)),
)
if err := remove.SetRequestPredicate(keys, pg.FieldMap{
	"document_id": "document_id",
	"line_no": "line_no",
}); err != nil {
	return err
}
if err := remove.SetAccessPolicy(access.Scope(access.Eq("customer_id", customerID)), pg.PolicyBinding{
	Rows: pg.FieldMap{"customer_id": "customer_id"},
}); err != nil {
	return err
}
params := []any{}
query, err := remove.BuildSQL(&params)
```

This becomes `(key_group_1 OR key_group_2) AND policy`, not an ungrouped mixture.
The builders do not execute transactions or decide batch all-or-nothing semantics.
The caller must inspect affected-row counts and implement any transactional batch
requirements. The custom delete path in the supplied `webapp` needs separate
integration; modifying `PgDelete` alone does not change it.

`CompilePredicate` returns an expression without WHERE, appending placeholders
after any existing parameters. Use it in carefully composed custom SQL. It does
not rewrite arbitrary queries or attach a principal. Never put raw request SQL in
`Relation()`, projected fields, aggregate expressions, or legacy `SetExpression`.
Those existing APIs remain trusted application SQL, not a new security boundary.

## Decisions and errors

Use `AllowAll()` for explicit unrestricted access, `DenyAll()` for denial, and
`Scope(...)` / `ForInsert(...)` for restricted access. Zero policies are invalid.
Empty `In` scopes compile to FALSE; zero predicates and empty AND/OR groups fail
validation. Use explicit `True()`/`False()` when a constant is intended.

A failed `SetAccessPolicy` or `SetRequestPredicate` is remembered by the builder.
Ignoring its error does not produce unprotected SQL. `BuildSQL` returns the error
without a partial query or modified parameter slice. Legacy `SQL` and
`CollectionSQL` panic on such errors, consistent with their lack of an error
return; prefer the new Build methods in policy-aware execution paths.

Useful error categories include `access.ErrDenied`, `access.ErrInvalidPolicy`,
`access.ErrInvalidField`, `access.ErrWriteConflict`, `pg.ErrMissingTarget`,
`pg.ErrDuplicateField`, and `modelbind.ErrMissingKey`. Use `errors.Is` and
`errors.As`. `access.FieldError` provides a logical field and stable reason code
without including the enforced value. These new errors expose untranslated codes
for application/UI translation; they are not HTTP status decisions.

Policies snapshot scalar values and mapping/rule maps on construction/installation.
Builders are still mutable, request-local objects: do not share one concurrently.
Discard a builder after a binding error, as with the existing binders.

## Integration boundary

This layer does not implement a user/role registry, resolve sessions, infer role
composition, install middleware, or modify codegen. A future `webapp` integration
must cover HTTP, WebSocket dispatch, direct service calls, jobs, both model/input
CRUD variants, custom deletes, exports, and notification recipients. Protected
resources must fail when principal/policy resolution fails. Never turn a lookup
error into `AllowAll()` or an omitted setter call.

`types.AccessScopedModel` lets related models declare the same resource:

```go
func (*OrderInput) AccessResource() string {
	return "Order"
}
```

The mapping to actual table/view/projection columns remains explicit per builder.
Do not parse a resource identity out of `Relation()`.

This implementation checks existing rows in SQL and supported submitted scalar
values in Go. It does **not** evaluate final database rows after triggers, verify
arbitrary cross-table constraints, provide an SQL EXISTS policy node, implement
upsert/MERGE, or replace native PostgreSQL RLS. Changes made by triggers/defaults,
related-table writes, and custom SQL need separate database/application controls.
For comparison, PostgreSQL's native USING and WITH CHECK distinction is documented
in its CREATE POLICY documentation; native checks also account for BEFORE-trigger
changes to new rows.

## Validation

On your normal Go 1.25+ installation:

```bash
go test ./...
go test -race ./...
go vet ./...
```

See `IMPLEMENTATION_NOTES.md` for the checks actually run while preparing this
patch and the local toolchain/PostgreSQL limitations.

PostgreSQL reference: `https://www.postgresql.org/docs/current/sql-createpolicy.html`.
