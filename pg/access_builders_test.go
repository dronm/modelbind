package pg

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/types"
)

func ownerBinding() PolicyBinding {
	return PolicyBinding{Rows: FieldMap{"owner": "customer_id"}, Writes: FieldMap{"owner": "customer_id"}}
}

func ownerUpdatePolicy() access.Policy {
	return access.Policy{Decision: access.Restricted, Rows: access.Eq("owner", 42), Writes: access.WriteRules{"owner": access.Enforce(42)}}
}

func TestCollectionPolicyGroupsLegacyORAndScopesTotals(t *testing.T) {
	filters := &PgFilters{}
	filters.Add("o", "status", "draft", types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)
	filters.Add("o", "urgent", true, types.SQLFilterOperatorEq, types.SQLFilterJoinOr)
	query := NewPgSelect(testModel{relation: "orders o"}, filters, nil, nil)
	query.AddField("o.id", nil)
	query.AddAggField("count(*)", nil)
	if err := query.SetAccessPolicy(access.Scope(access.Eq("owner", 42)), PolicyBinding{Rows: FieldMap{"owner": "o.customer_id"}}); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	data, aggregate, err := query.BuildCollectionSQL(&params)
	where := " WHERE ((o.status = $1) OR (o.urgent = $2)) AND (o.customer_id = $3)"
	if err != nil || data != "SELECT o.id FROM orders o"+where || aggregate != "SELECT count(*) FROM orders o"+where {
		t.Fatalf("bad scoped collection: %q %q %v", data, aggregate, err)
	}
	if !reflect.DeepEqual(params, []any{"draft", true, int64(42)}) {
		t.Fatalf("bad shared parameters: %#v", params)
	}
	// Replacing or clearing caller filters cannot remove the policy.
	var empty *PgFilters
	if err := query.SetFilter(empty); err != nil {
		t.Fatal(err)
	}
	params = nil
	data, err = query.BuildSQL(&params)
	if err != nil || data != "SELECT o.id FROM orders o WHERE (o.customer_id = $1)" {
		t.Fatalf("policy disappeared with caller filters: %q %v", data, err)
	}
}

func TestDetailPolicyIsIndependentOfProjectionFields(t *testing.T) {
	filter := &PgFilters{}
	filter.Add("", "id", 10, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)
	query := NewPgDetailSelect(testModel{relation: "orders"}, filter)
	query.AddField("id", nil) // customer_id is deliberately NOT in the projection.
	if err := query.SetAccessPolicy(access.Scope(access.Eq("owner", 42)), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	sql, err := query.BuildSQL(&params)
	if err != nil || sql != "SELECT id FROM orders WHERE (id = $1) AND (customer_id = $2)" || !reflect.DeepEqual(params, []any{10, int64(42)}) {
		t.Fatalf("bad detail scope: %q %#v %v", sql, params, err)
	}
}

func TestRequestPredicateAndPolicyRemainSeparate(t *testing.T) {
	query := NewPgDetailSelect(testModel{relation: "orders"}, nil)
	query.AddField("id", nil)
	if err := query.SetRequestPredicate(access.Or(access.Eq("id", 1), access.Eq("id", 2)), FieldMap{"id": "id"}); err != nil {
		t.Fatal(err)
	}
	if err := query.SetAccessPolicy(access.Scope(access.In("owner", 42, 43)), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	params := []any{"prefix"}
	sql, err := query.BuildSQL(&params)
	want := "SELECT id FROM orders WHERE ((id = $2) OR (id = $3)) AND (customer_id IN ($4,$5))"
	if err != nil || sql != want || !reflect.DeepEqual(params, []any{"prefix", int64(1), int64(2), int64(42), int64(43)}) {
		t.Fatalf("bad predicate composition: %q %#v %v", sql, params, err)
	}
}

func TestUpdatePolicyChecksOwnershipAndNewValue(t *testing.T) {
	for _, field := range []struct {
		name    string
		present bool
		value   any
		wantErr error
	}{
		{"absent", false, nil, nil},
		{"equal", true, int32(42), nil},
		{"different", true, 99, access.ErrWriteConflict},
		{"null", true, nil, access.ErrWriteConflict},
	} {
		t.Run(field.name, func(t *testing.T) {
			query := NewPgUpdate(testModel{relation: "orders"})
			query.AddField("name", "changed")
			query.Filter().Add("", "id", 10, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)
			if err := query.SetAccessPolicy(ownerUpdatePolicy(), ownerBinding()); err != nil {
				t.Fatal(err)
			}
			if field.present {
				query.AddField("customer_id", field.value) // Deliberately added AFTER installing policy.
			}
			params := []any{"prefix"}
			sql, err := query.BuildSQL(&params)
			if !errors.Is(err, field.wantErr) {
				t.Fatalf("error = %v; want %v", err, field.wantErr)
			}
			if err != nil {
				if sql != "" || !reflect.DeepEqual(params, []any{"prefix"}) {
					t.Fatal("rejected update leaked partial SQL/params")
				}
				return
			}
			if !strings.Contains(sql, "AND (customer_id = $") {
				t.Fatalf("missing row ownership restriction: %q", sql)
			}
			if !field.present {
				want := "UPDATE orders SET name = $2 WHERE (id = $3) AND (customer_id = $4)"
				if sql != want || !reflect.DeepEqual(params, []any{"prefix", "changed", 10, int64(42)}) {
					t.Fatalf("absent owner must not be added to SET: %q %#v", sql, params)
				}
			}
		})
	}
}

func TestInsertPolicyAddsServerOnlyColumn(t *testing.T) {
	query := NewPgInsert(testModel{relation: "orders"})
	query.AddField("name", "new order")
	query.AddRetField("id", nil)
	rules := access.WriteRules{"owner": access.Enforce(42), "status": access.Default("draft")}
	binding := PolicyBinding{Writes: FieldMap{"owner": "customer_id", "status": "status"}}
	if err := query.SetAccessPolicy(access.ForInsert(rules), binding); err != nil {
		t.Fatal(err)
	}
	// Mutating the caller's registration maps cannot change an installed policy.
	rules["owner"] = access.Enforce(99)
	binding.Writes["owner"] = "other_customer_id"
	params := []any{}
	sql, err := query.BuildSQL(&params)
	want := "INSERT INTO orders (name,customer_id,status) VALUES ($1,$2,$3) RETURNING id"
	if err != nil || sql != want || !reflect.DeepEqual(params, []any{"new order", int64(42), "draft"}) {
		t.Fatalf("bad insert: %q %#v %v", sql, params, err)
	}
	if query.InsertFieldLen() != 1 {
		t.Fatal("building SQL must not mutate caller field list")
	}
	params = nil
	second, err := query.BuildSQL(&params)
	if err != nil || second != sql || len(params) != 3 {
		t.Fatalf("repeated build added duplicate policy fields: %q %#v %v", second, params, err)
	}
}

func TestInsertAllowedSetAndLateOverride(t *testing.T) {
	query := NewPgInsert(testModel{relation: "orders"})
	if err := query.SetAccessPolicy(access.ForInsert(access.WriteRules{"site": access.Allowed(3, 8)}), PolicyBinding{Writes: FieldMap{"site": "site_id"}}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{nil, 12, "3"} {
		if err := query.SetField("site_id", value); err != nil {
			t.Fatal(err)
		}
		params := []any{}
		if sql, err := query.BuildSQL(&params); !errors.Is(err, access.ErrWriteConflict) || sql != "" {
			t.Fatalf("invalid site accepted: %q %v", sql, err)
		}
	}
	if err := query.SetField("site_id", 8); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	if _, err := query.BuildSQL(&params); err != nil {
		t.Fatal(err)
	}
	missing := NewPgInsert(testModel{relation: "orders"})
	if err := missing.SetAccessPolicy(access.ForInsert(access.WriteRules{"site": access.Allowed(3, 8)}), PolicyBinding{Writes: FieldMap{"site": "site_id"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := missing.BuildSQL(&params); !errors.Is(err, access.ErrWriteConflict) {
		t.Fatal("membership must not choose an arbitrary missing insert value")
	}
}

func TestDeleteCompositeKeyGroups(t *testing.T) {
	query := NewPgDelete(testModel{relation: "document_items"}, nil)
	keys := access.Or(
		access.And(access.Eq("document", 1), access.Eq("line", 2)),
		access.And(access.Eq("document", 3), access.Eq("line", 4)),
	)
	if err := query.SetRequestPredicate(keys, FieldMap{"document": "document_id", "line": "line_no"}); err != nil {
		t.Fatal(err)
	}
	if err := query.SetAccessPolicy(access.Scope(access.Eq("owner", 42)), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	sql, err := query.BuildSQL(&params)
	want := "DELETE FROM document_items WHERE (((document_id = $1) AND (line_no = $2)) OR ((document_id = $3) AND (line_no = $4))) AND (customer_id = $5)"
	if err != nil || sql != want || !reflect.DeepEqual(params, []any{int64(1), int64(2), int64(3), int64(4), int64(42)}) {
		t.Fatalf("bad composite delete: %q %#v %v", sql, params, err)
	}
}

func TestProtectedWritesRequireIndependentTarget(t *testing.T) {
	update := NewPgUpdate(testModel{relation: "orders"})
	update.AddField("name", "x")
	if err := update.SetAccessPolicy(ownerUpdatePolicy(), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	if _, err := update.BuildSQL(&params); !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("policy alone accepted as key: %v", err)
	}
	for _, predicate := range []access.Predicate{access.True(), access.Or(access.Eq("id", 1), access.True())} {
		if err := update.SetRequestPredicate(predicate, FieldMap{"id": "id"}); err != nil {
			t.Fatal(err)
		}
		if _, err := update.BuildSQL(&params); !errors.Is(err, ErrMissingTarget) {
			t.Fatalf("unrestricted caller predicate accepted: %v", err)
		}
	}
	remove := NewPgDelete(testModel{relation: "orders"}, nil)
	if err := remove.SetAccessPolicy(access.AllowAll(), PolicyBinding{}); err != nil {
		t.Fatal(err)
	}
	if _, err := remove.BuildSQL(&params); !errors.Is(err, ErrMissingTarget) {
		t.Fatal("even explicitly unrestricted protected deletes require a target")
	}
	if err := remove.SetRequestPredicate(access.In("id"), FieldMap{"id": "id"}); err != nil {
		t.Fatal(err)
	}
	if sql, err := remove.BuildSQL(&params); err != nil || sql != "DELETE FROM orders WHERE (FALSE)" {
		t.Fatalf("empty caller key set should be a safe no-op: %q %v", sql, err)
	}
}

func TestSetFieldReplacesDuplicatesAndPolicyRechecks(t *testing.T) {
	insert := NewPgInsert(testModel{relation: "orders"})
	insert.AddField("name", "order")
	insert.AddField("customer_id", 7)
	insert.AddField("Customer_ID", 8)
	insert.AddField("status", "draft")
	if err := insert.SetField("customer_id", 42); err != nil {
		t.Fatal(err)
	}
	if insert.InsertFieldLen() != 3 {
		t.Fatal("duplicate assignment not collapsed")
	}
	if err := insert.SetAccessPolicy(access.ForInsert(access.WriteRules{"owner": access.Enforce(42)}), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	sql, err := insert.BuildSQL(&params)
	if err != nil || sql != "INSERT INTO orders (name,customer_id,status) VALUES ($1,$2,$3)" {
		t.Fatalf("replacement changed order or duplicated field: %q %v", sql, err)
	}
	if err := insert.SetField("customer_id", 99); err != nil {
		t.Fatal(err)
	}
	if _, err := insert.BuildSQL(&params); !errors.Is(err, access.ErrWriteConflict) {
		t.Fatal("late SetField bypassed access policy")
	}
	update := NewPgUpdate(testModel{relation: "orders"})
	update.AddField("name", "old")
	update.AddField("NAME", "duplicate")
	if err := update.SetField("name", "new"); err != nil || update.AssignerLen() != 1 {
		t.Fatalf("update replacement failed: %v", err)
	}
	if err := update.SetField("name;drop table orders", "x"); err == nil || update.AssignerLen() != 1 {
		t.Fatal("unsafe SetField modified assigners")
	}
	if err := insert.SetField("o.customer_id", 42); err == nil {
		t.Fatal("qualified write assignment accepted")
	}
}

func TestProtectedBuilderRejectsDuplicateWrites(t *testing.T) {
	query := NewPgInsert(testModel{relation: "orders"})
	query.AddField("customer_id", 42)
	query.AddField("CUSTOMER_ID", 99)
	if err := query.SetAccessPolicy(access.ForInsert(access.WriteRules{"owner": access.Enforce(42)}), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	if _, err := query.BuildSQL(&params); !errors.Is(err, ErrDuplicateField) {
		t.Fatalf("duplicate columns accepted: %v", err)
	}
}

type protectedBuilder interface {
	SetAccessPolicy(access.Policy, PolicyBinding) error
	BuildSQL(*[]any) (string, error)
	SQL(*[]any) string
}

func TestDeniedAndInvalidPoliciesFailClosedForEveryBuilder(t *testing.T) {
	factories := map[string]func() protectedBuilder{
		"list":   func() protectedBuilder { return NewPgSelect(testModel{relation: "orders"}, nil, nil, nil) },
		"detail": func() protectedBuilder { return NewPgDetailSelect(testModel{relation: "orders"}, nil) },
		"insert": func() protectedBuilder { return NewPgInsert(testModel{relation: "orders"}) },
		"update": func() protectedBuilder { return NewPgUpdate(testModel{relation: "orders"}) },
		"delete": func() protectedBuilder { q := NewPgDelete(testModel{relation: "orders"}, nil); return &q },
	}
	for name, factory := range factories {
		for _, decision := range []struct {
			name   string
			policy access.Policy
			want   error
		}{
			{"denied", access.DenyAll(), access.ErrDenied},
			{"unresolved", access.Policy{}, access.ErrInvalidPolicy},
		} {
			t.Run(name+"/"+decision.name, func(t *testing.T) {
				query := factory()
				if err := query.SetAccessPolicy(decision.policy, PolicyBinding{}); !errors.Is(err, decision.want) {
					t.Fatalf("installation error = %v", err)
				}
				params := []any{"keep"}
				sql, err := query.BuildSQL(&params)
				if !errors.Is(err, decision.want) || sql != "" || !reflect.DeepEqual(params, []any{"keep"}) {
					t.Fatalf("ignored setter error weakened query: %q %#v %v", sql, params, err)
				}
				defer func() {
					if recover() == nil {
						t.Fatal("legacy SQL must panic instead of dropping failed policy")
					}
				}()
				_ = query.SQL(&params)
			})
		}
	}
}

func TestMappingErrorsFailClosed(t *testing.T) {
	for _, binding := range []PolicyBinding{
		{},
		{Rows: FieldMap{"owner": "customer_id"}},
		{Rows: FieldMap{"owner": "customer_id"}, Writes: FieldMap{"owner": "o.customer_id"}},
		{Rows: FieldMap{"owner": "id) OR TRUE --"}, Writes: FieldMap{"owner": "customer_id"}},
	} {
		query := NewPgUpdate(testModel{relation: "orders"})
		if err := query.SetAccessPolicy(ownerUpdatePolicy(), binding); err == nil {
			t.Fatal("invalid binding accepted")
		}
		params := []any{}
		if sql, err := query.BuildSQL(&params); err == nil || sql != "" {
			t.Fatal("invalid binding was ignored")
		}
	}
	query := NewPgInsert(testModel{relation: "orders"})
	policy := access.ForInsert(access.WriteRules{"owner": access.Enforce(42), "other": access.Enforce(7)})
	if err := query.SetAccessPolicy(policy, PolicyBinding{Writes: FieldMap{"owner": "customer_id", "other": "CUSTOMER_ID"}}); err == nil {
		t.Fatal("conflicting aliases for a write column accepted")
	}
}

func TestInvalidRequestAndUnsafeLegacyBuildRollback(t *testing.T) {
	query := NewPgDetailSelect(testModel{relation: "orders"}, nil)
	query.AddField("id", nil)
	if err := query.SetRequestPredicate(access.Predicate{}, nil); err == nil {
		t.Fatal("zero request predicate accepted")
	}
	params := []any{"keep"}
	if sql, err := query.BuildSQL(&params); err == nil || sql != "" || len(params) != 1 {
		t.Fatal("failed request predicate was ignored")
	}
	insert := NewPgInsert(testModel{relation: "orders"})
	insert.AddField("name", "first")
	insert.AddField("x;DROP TABLE orders", "second")
	if sql, err := insert.BuildSQL(&params); err == nil || sql != "" || !reflect.DeepEqual(params, []any{"keep"}) {
		t.Fatalf("BuildSQL did not roll back parameters: %q %#v %v", sql, params, err)
	}
}

func TestEmptyScopeAndPolicyRuleSnapshot(t *testing.T) {
	query := NewPgSelect(testModel{relation: "orders"}, nil, nil, nil)
	query.AddField("id", nil)
	query.AddAggField("count(*)", nil)
	if err := query.SetAccessPolicy(access.Scope(access.In("owner")), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	sql, aggregate, err := query.BuildCollectionSQL(&params)
	if err != nil || !strings.HasSuffix(sql, "WHERE (FALSE)") || !strings.HasSuffix(aggregate, "WHERE (FALSE)") || len(params) != 0 {
		t.Fatalf("empty scope broadened access: %q %q %#v %v", sql, aggregate, params, err)
	}
	insert := NewPgInsert(testModel{relation: "orders"})
	if _, _, err := insert.AccessWriteRules(); !errors.Is(err, access.ErrInvalidPolicy) {
		t.Fatal("missing policy must not be treated as unrestricted by aware binding")
	}
	if err := insert.SetAccessPolicy(access.ForInsert(access.WriteRules{"owner": access.Enforce(42)}), ownerBinding()); err != nil {
		t.Fatal(err)
	}
	_, rules, err := insert.AccessWriteRules()
	if err != nil {
		t.Fatal(err)
	}
	rules["owner"] = access.Enforce(99)
	params = nil
	if _, err := insert.BuildSQL(&params); err != nil || !reflect.DeepEqual(params, []any{int64(42)}) {
		t.Fatalf("returned rules changed installed policy: %#v %v", params, err)
	}
}
