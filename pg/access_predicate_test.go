package pg

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dronm/modelbind/access"
)

func TestCompilePredicate(t *testing.T) {
	fields := FieldMap{"owner": "o.customer_id", "id": "o.id", "status": "o.status", "reference": "o.reference->>'id'"}
	tests := []struct {
		name       string
		predicate  access.Predicate
		wantSQL    string
		wantParams []any
	}{
		{"true", access.True(), "TRUE", nil},
		{"false", access.False(), "FALSE", nil},
		{"equality", access.Eq("owner", 42), "o.customer_id = $2", []any{int64(42)}},
		{"null", access.Eq("owner", nil), "o.customer_id IS NULL", nil},
		{"not_null", access.NotEq("owner", nil), "o.customer_id IS NOT NULL", nil},
		{"in", access.In("id", 3, 8), "o.id IN ($2,$3)", []any{int64(3), int64(8)}},
		{"empty_set", access.In("id"), "FALSE", nil},
		{"json_reference", access.Eq("reference", "100"), "o.reference->>'id' = $2", []any{"100"}},
		{"nested", access.And(access.Or(access.Eq("status", "draft"), access.Eq("status", "new")), access.Eq("owner", 42)), "((o.status = $2) OR (o.status = $3)) AND (o.customer_id = $4)", []any{"draft", "new", int64(42)}},
		{"membership_or_null", access.Or(access.In("owner", 3, 8), access.IsNull("owner")), "(o.customer_id IN ($2,$3)) OR (o.customer_id IS NULL)", []any{int64(3), int64(8)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := []any{"existing"}
			query, err := CompilePredicate(tt.predicate, fields, &params)
			if err != nil || query != tt.wantSQL {
				t.Fatalf("query = %q, error = %v; want %q", query, err, tt.wantSQL)
			}
			want := append([]any{"existing"}, tt.wantParams...)
			if !reflect.DeepEqual(params, want) {
				t.Fatalf("params = %#v; want %#v", params, want)
			}
		})
	}
}

func TestPredicateCompilerRejectsUnsafeConfigurationAtomically(t *testing.T) {
	tests := []struct {
		predicate access.Predicate
		fields    FieldMap
	}{
		{access.Predicate{}, nil},
		{access.Eq("id", 1), nil},
		{access.In("id"), nil}, // Even empty sets must have a valid explicit mapping.
		{access.Eq("id", 1), FieldMap{"id": "id) OR TRUE --"}},
		{access.Eq("id", 1), FieldMap{"id": "o.id", "bad;field": "o.name"}},
		{access.Compare("id", "= $1 OR TRUE --", 1), FieldMap{"id": "id"}},
		{access.And(access.Eq("id", 1), access.Eq("owner", 2)), FieldMap{"id": "id"}},
	}
	for _, tt := range tests {
		params := []any{"original"}
		query, err := CompilePredicate(tt.predicate, tt.fields, &params)
		if err == nil || query != "" || !reflect.DeepEqual(params, []any{"original"}) {
			t.Fatalf("invalid compilation was not atomic: %q %#v %v", query, params, err)
		}
	}
	if _, err := CompilePredicate(access.True(), nil, nil); err == nil {
		t.Fatal("nil parameter pointer should fail")
	}
}

func TestPredicateDoesNotInterpolateValues(t *testing.T) {
	value := "' OR TRUE; DROP TABLE orders; --"
	params := []any{}
	query, err := CompilePredicate(access.Eq("name", value), FieldMap{"name": "name"}, &params)
	if err != nil || query != "name = $1" || strings.Contains(query, value) || !reflect.DeepEqual(params, []any{value}) {
		t.Fatalf("unsafe parameter handling: %q %#v %v", query, params, err)
	}
	_, err = CompilePredicate(access.Eq("name", value), FieldMap{}, &params)
	if !errors.Is(err, access.ErrInvalidField) {
		t.Fatal("expected missing field map error")
	}
}

func FuzzCompilePredicateParameterization(f *testing.F) {
	f.Add("name", "' OR TRUE --")
	f.Add("o.name", "normal")
	f.Add("name);DROP TABLE orders;--", "$17")
	f.Fuzz(func(t *testing.T, column, value string) {
		params := []any{"prefix"}
		sql, err := CompilePredicate(access.Eq("field", value), FieldMap{"field": column}, &params)
		if err != nil {
			if sql != "" || !reflect.DeepEqual(params, []any{"prefix"}) {
				t.Fatal("failed compilation changed parameters")
			}
			return
		}
		if len(params) != 2 || params[1] != value || !strings.HasSuffix(sql, " = $2") {
			t.Fatalf("broken parameterization: %q %#v", sql, params)
		}
	})
}
