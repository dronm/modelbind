package pg

import (
	"reflect"
	"testing"

	"github.com/dronm/modelbind/types"
)

func TestFilterSQLDefaultOperator(t *testing.T) {
	params := []any{}
	filter := NewPgFilter("id", int64(10))

	gotSQL := filter.SQL(&params)
	wantSQL := "id = $1"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{int64(10)}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestFilterSQLWithFieldPrefix(t *testing.T) {
	params := []any{}
	filter := PgFilter{
		fieldID:   "id",
		fieldPref: "u",
		value:     10,
		operator:  types.SQLFilterOperatorGrEq,
	}

	gotSQL := filter.SQL(&params)
	wantSQL := "u.id >= $1"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
}

func TestFilterSQLExpressionWithParam(t *testing.T) {
	params := []any{"existing"}
	filter := PgFilter{
		value:      "hello",
		expression: "to_tsvector(name) @@ plainto_tsquery({{PARAM}})",
	}

	gotSQL := filter.SQL(&params)
	wantSQL := "to_tsvector(name) @@ plainto_tsquery($2)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{"existing", "hello"}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestFilterSQLExpressionWithoutParam(t *testing.T) {
	params := []any{}
	filter := PgFilter{expression: "deleted_at IS NULL"}

	gotSQL := filter.SQL(&params)
	wantSQL := "deleted_at IS NULL"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
	if len(params) != 0 {
		t.Fatalf("expected no params, got %#v", params)
	}
}

func TestFilterSQLNullOperators(t *testing.T) {
	tests := []struct {
		name     string
		operator types.SQLFilterOperator
		wantSQL  string
	}{
		{name: "is null", operator: types.SQLFilterOperatorIs, wantSQL: "deleted_at IS NULL"},
		{name: "is not null", operator: types.SQLFilterOperatorIn, wantSQL: "deleted_at IS NOT NULL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := []any{}
			filter := PgFilter{fieldID: "deleted_at", operator: test.operator}

			gotSQL := filter.SQL(&params)
			if gotSQL != test.wantSQL {
				t.Fatalf("expected SQL %q, got %q", test.wantSQL, gotSQL)
			}
			if len(params) != 0 {
				t.Fatalf("expected no params, got %#v", params)
			}
		})
	}
}

func TestFilterSQLAnyOperator(t *testing.T) {
	params := []any{}
	filter := PgFilter{fieldID: "id", value: []int{1, 2}, operator: types.SQLFilterOperatorAny}

	gotSQL := filter.SQL(&params)
	wantSQL := "id = ANY($1)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{[]int{1, 2}}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestFiltersSQL(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.Add("u", "name", "Alice", types.SQLFilterOperatorILk, types.SQLFilterJoinAnd)
	filters.Add("u", "age", 18, types.SQLFilterOperatorGrEq, types.SQLFilterJoinOr)

	gotSQL := filters.SQL(&params)
	wantSQL := " WHERE (u.name ILIKE $1) OR (u.age >= $2)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{"Alice", 18}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}

	if filters.Len() != 2 {
		t.Fatalf("expected len 2, got %d", filters.Len())
	}
}

func TestFiltersSQLDefaultsJoinToAnd(t *testing.T) {
	params := []any{}
	filters := PgFilters{
		{fieldID: "name", value: "Alice"},
		{fieldID: "age", value: 18, operator: types.SQLFilterOperatorGrEq},
	}

	gotSQL := filters.SQL(&params)
	wantSQL := " WHERE (name = $1) AND (age >= $2)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
}

func TestFiltersSQLEmpty(t *testing.T) {
	params := []any{}
	filters := PgFilters{}

	gotSQL := filters.SQL(&params)
	if gotSQL != "" {
		t.Fatalf("expected empty SQL, got %q", gotSQL)
	}
	if len(params) != 0 {
		t.Fatalf("expected no params, got %#v", params)
	}
}

func TestFiltersAddFullTextSearch(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.AddFullTextSearch("p", "search_vector", "boots & gloves", types.SQLFilterJoinAnd)

	gotSQL := filters.SQL(&params)
	wantSQL := " WHERE p.search_vector @@ to_tsquery('russian', $1)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{"boots & gloves"}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestFiltersAddArrayInclude(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.AddArrayInclude("p", "tag_ids", []int{1, 2}, types.SQLFilterJoinAnd)

	gotSQL := filters.SQL(&params)
	wantSQL := " WHERE p.tag_ids = ANY($1)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
}

func TestFiltersAddColumnArrayInclude(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.AddColumnArrayInclude("p", "tag_ids", 5, types.SQLFilterJoinAnd)

	gotSQL := filters.SQL(&params)
	wantSQL := " WHERE $1 = ANY(p.tag_ids)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
}

func TestFilterSQLPanicsOnUnsafeField(t *testing.T) {
	params := []any{}
	filter := PgFilter{fieldID: "id;DROP TABLE users", value: 10}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unsafe field reference")
		}
	}()

	_ = filter.SQL(&params)
}
