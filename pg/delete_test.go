package pg

import (
	"reflect"
	"testing"

	"github.com/dronm/modelbind/types"
)

func TestDeleteSQL(t *testing.T) {
	params := []any{}
	filters := PgFilters{}
	filters.Add("", "id", 10, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)
	deleteQuery := NewPgDelete(testModel{relation: "users"}, filters)

	gotSQL := deleteQuery.SQL(&params)
	wantSQL := "DELETE FROM users WHERE id = $1"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{10}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestDeleteSQLWithoutFilter(t *testing.T) {
	params := []any{}
	deleteQuery := NewPgDelete(testModel{relation: "users"}, PgFilters{})

	gotSQL := deleteQuery.SQL(&params)
	wantSQL := "DELETE FROM users"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
}

func TestDeleteModelAndFilter(t *testing.T) {
	filters := PgFilters{{fieldID: "id", value: 10}}
	deleteQuery := NewPgDelete(testModel{relation: "users"}, filters)

	if deleteQuery.Model().Relation() != "users" {
		t.Fatalf("expected relation users, got %q", deleteQuery.Model().Relation())
	}
	if deleteQuery.Filter().Len() != 1 {
		t.Fatalf("expected filter len 1, got %d", deleteQuery.Filter().Len())
	}
}
