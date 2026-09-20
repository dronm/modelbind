# Access-policy implementation notes

## Scope of this patch

Baseline: the uploaded `modelbind(3).zip` source tree. `webapp`, `codegen`, and
application schemas/migrations are not changed. The module path and Go 1.25.0
requirement are unchanged; no third-party dependency was added.

New functionality is opt-in. Existing constructor and interface signatures are
preserved. Normal successful legacy SQL without a policy/request predicate retains
its prior formatting. Existing `AddField` methods still append; new replacement
methods are separate. New `BuildSQL` methods return errors; legacy SQL methods
panic rather than return an unprotected query after a failed policy setup.

New files implement the independent `access` policy vocabulary, PostgreSQL
predicate/policy adapters, typed effective-input preparation, optional integration
interfaces, regression tests, executable examples, and usage documentation.
Existing concrete SELECT/detail/INSERT/UPDATE/DELETE renderers now support the
additional state; `AbsentFieldSet` gains `Clone` and `SetPresent`.

README corrections also align its collection constructor example and default
value-list separator with the existing implementation, and clarify that relation
strings/SQL projections remain trusted application inputs.

## Verification actually performed

The execution environment contains **Go 1.23.2**, not the module's Go 1.25 target.
Downloading the requested Go 1.25 toolchain failed because network access from the
execution environment was unavailable. The original module's Go version was NOT
lowered in the deliverable.

Verification used an isolated source copy with exactly these compatibility edits:

1. The copy's `go.mod` directive was changed from `go 1.25.0` to `go 1.23.0`.
2. The pre-existing `strings.SplitSeq` loop in `input_decode.go` was changed in
   that copy to the equivalent `for _, part := range strings.Split(...)` loop.

Neither compatibility edit is part of the patch. All other Go source in the test
copy was checked against the delivered source byte-for-byte.

Results in that compatibility copy:

| Check | Result |
| --- | --- |
| `go test -count=1 -json ./...` | Passed, including all pre-existing tests and new tests/examples/fuzz seeds. |
| `go test -race -shuffle=on -count=3 ./...` | Passed. |
| `go vet ./...` | Passed. |
| `go test ./pg -run='^$' -fuzz=FuzzCompilePredicateParameterization -fuzztime=5s -parallel=2` | Passed, 145,648 fuzz executions in the recorded run. |

This is not a claim of a native Go 1.25 test run. No live PostgreSQL server/client
was available, so SQL generation and policy behavior were tested at the Go level,
not through database execution. Run the normal test/vet/race commands on your Go
1.25+ installation, and add application/database integration tests when wiring
`webapp` to these APIs.

The regression suite covers grouped legacy OR filters, detail reads, aggregate
scope consistency, nested/composite caller predicates, parameter offsets,
parameterization, invalid/missing mappings, empty permitted sets, denied/unresolved
policies, ignored setup errors, input ownership conflicts, explicit NULL, missing
keys, immutable/allowed/default/enforced write semantics, safe assignment
replacement, late mutation attempts, numeric precision/overflow, named enum
values, request-presence preservation, RETURNING targets, and policy snapshots.

## Integration work still belongs in webapp

Resolve user/role policy decisions from trusted identity data. Mark protected
resources explicitly and reject missing identity, failed policy resolution, or
missing policy configuration before creating/executing queries. Installing an
optional model interface alone is not enforcement.

Install policies across HTTP and WebSocket paths, direct service calls and jobs,
all CRUD variants, custom deletes, projections, totals, reports and exports.
Route permissions remain independent of row restrictions.

Prepare server-supplied input before required-field validation. Policy-aware bind
helpers return a separate effective model; use it for RETURNING and responses.
Do not keep returning the original constructor model by mistake.

Keep key validation independent of row scope and handle affected-row/no-row
results explicitly. Batch all-or-nothing behavior, transaction management, and
notification-recipient authorization are not supplied by a SQL builder.

The initial write-rule implementation supports scalar rules, not arbitrary final
row validation after database triggers, cross-table membership queries, native
RLS, upserts, or MERGE. See `ACCESS_POLICIES.md` for the exact contract.

Suggested commit subject:

```text
feat: add resolved row policies and scalar write-rule enforcement
```
