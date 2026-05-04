package pg

import (
	"reflect"
	"testing"

	"github.com/dronm/modelbind/types"
)

func TestSelectSQL(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.Add("u", "age", 18, types.SQLFilterOperatorGrEq, types.SQLFilterJoinAnd)
	sorters := PgSorters{{fieldID: "name", direct: types.SQLSortAsc, fieldPref: "u"}}
	limit := NewPgLimit(20, 10)
	selectQuery := NewPgSelect(testModel{relation: "users u"}, &filters, &sorters, limit)
	selectQuery.AddField("u.id", nil)
	selectQuery.AddField("u.name", nil)

	gotSQL := selectQuery.SQL(&params)
	wantSQL := "SELECT u.id,u.name FROM users u WHERE u.age >= $1 ORDER BY u.name ASC OFFSET 20 LIMIT 10"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{18}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestSelectCollectionSQL(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.Add("", "active", true, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)
	selectQuery := NewPgSelect(testModel{relation: "users"}, &filters, nil, nil)
	selectQuery.AddField("id", nil)
	selectQuery.AddField("name", nil)
	selectQuery.AddAggField("count(*)", nil)

	gotSQL, gotAggSQL := selectQuery.CollectionSQL(&params)
	wantSQL := "SELECT id,name FROM users WHERE active = $1"
	wantAggSQL := "SELECT count(*) FROM users WHERE active = $1"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
	if gotAggSQL != wantAggSQL {
		t.Fatalf("expected aggregate SQL %q, got %q", wantAggSQL, gotAggSQL)
	}

	wantParams := []any{true}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestSelectCollectionSQLWithoutAggFields(t *testing.T) {
	params := []any{}
	selectQuery := NewPgSelect(testModel{relation: "users"}, nil, nil, nil)
	selectQuery.AddField("id", nil)

	gotSQL, gotAggSQL := selectQuery.CollectionSQL(&params)
	wantSQL := "SELECT id FROM users"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
	if gotAggSQL != "" {
		t.Fatalf("expected empty aggregate SQL, got %q", gotAggSQL)
	}
}

func TestSelectAccessorsAndSetFilter(t *testing.T) {
	filters := PgFilters{}
	sorters := PgSorters{}
	limit := NewPgLimit(0, 10)
	selectQuery := NewPgSelect(testModel{relation: "users"}, &filters, &sorters, limit)
	selectQuery.AddField("id", "scan-id")

	if selectQuery.Model().Relation() != "users" {
		t.Fatalf("expected relation users, got %q", selectQuery.Model().Relation())
	}
	if selectQuery.Filter() == nil {
		t.Fatal("expected non-nil filter")
	}
	if selectQuery.Sorter() == nil {
		t.Fatal("expected non-nil sorter")
	}
	if selectQuery.Limit() == nil {
		t.Fatal("expected non-nil limit")
	}

	wantValues := []any{"scan-id"}
	if !reflect.DeepEqual(selectQuery.FieldValues(), wantValues) {
		t.Fatalf("expected field values %#v, got %#v", wantValues, selectQuery.FieldValues())
	}

	newFilters := PgFilters{{fieldID: "id", value: 10}}
	if err := selectQuery.SetFilter(&newFilters); err != nil {
		t.Fatalf("expected set filter success, got error: %v", err)
	}
	if selectQuery.Filter().Len() != 1 {
		t.Fatalf("expected filter len 1, got %d", selectQuery.Filter().Len())
	}
}

func TestSelectSetFilterRejectsWrongType(t *testing.T) {
	selectQuery := NewPgSelect(testModel{relation: "users"}, nil, nil, nil)

	err := selectQuery.SetFilter(wrongFilters{})
	if err == nil {
		t.Fatal("expected error")
	}
}
