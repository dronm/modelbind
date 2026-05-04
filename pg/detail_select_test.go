package pg

import (
	"reflect"
	"testing"

	"github.com/dronm/modelbind/types"
)

func TestDetailSelectSQL(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.Add("", "id", 10, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)
	selectQuery := NewPgDetailSelect(testModel{relation: "users"}, &filters)
	selectQuery.AddField("id", nil)
	selectQuery.AddField("name", nil)

	gotSQL := selectQuery.SQL(&params)
	wantSQL := "SELECT id,name FROM users WHERE id = $1"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{10}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestDetailSelectSQLWithoutFilter(t *testing.T) {
	params := []any{}
	selectQuery := NewPgDetailSelect(testModel{relation: "users"}, nil)
	selectQuery.AddField("id", "scan-id")

	gotSQL := selectQuery.SQL(&params)
	wantSQL := "SELECT id FROM users"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantValues := []any{"scan-id"}
	if !reflect.DeepEqual(selectQuery.FieldValues(), wantValues) {
		t.Fatalf("expected field values %#v, got %#v", wantValues, selectQuery.FieldValues())
	}
}

func TestDetailSelectAccessorsAndSetFilter(t *testing.T) {
	filters := PgFilters{}
	selectQuery := NewPgDetailSelect(testModel{relation: "users"}, &filters)

	if selectQuery.Model().Relation() != "users" {
		t.Fatalf("expected relation users, got %q", selectQuery.Model().Relation())
	}
	if selectQuery.Filter() == nil {
		t.Fatal("expected non-nil filter")
	}

	newFilters := PgFilters{{fieldID: "id", value: 10}}
	if err := selectQuery.SetFilter(&newFilters); err != nil {
		t.Fatalf("expected set filter success, got error: %v", err)
	}
	if selectQuery.Filter().Len() != 1 {
		t.Fatalf("expected filter len 1, got %d", selectQuery.Filter().Len())
	}
}

func TestDetailSelectSetFilterRejectsWrongType(t *testing.T) {
	selectQuery := NewPgDetailSelect(testModel{relation: "users"}, nil)

	err := selectQuery.SetFilter(wrongFilters{})
	if err == nil {
		t.Fatal("expected error")
	}
}
