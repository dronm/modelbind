package pg

import (
	"reflect"
	"testing"

	"github.com/dronm/modelbind/types"
)

func TestUpdateSQL(t *testing.T) {
	params := []any{}
	update := NewPgUpdate(testModel{relation: "users"})
	update.AddField("name", "Alice")
	update.AddField("age", 30)
	update.filter.Add("", "id", 10, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)

	gotSQL := update.SQL(&params)
	wantSQL := "UPDATE users SET name = $1, age = $2 WHERE id = $3"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{"Alice", 30, 10}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}

	if update.AssignerLen() != 2 {
		t.Fatalf("expected assigner len 2, got %d", update.AssignerLen())
	}
}

func TestUpdateSQLWithNilFilterAndAssigner(t *testing.T) {
	params := []any{}
	update := PgUpdate{model: testModel{relation: "users"}}

	gotSQL := update.SQL(&params)
	wantSQL := "UPDATE users SET "
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	if update.AssignerLen() != 0 {
		t.Fatalf("expected assigner len 0, got %d", update.AssignerLen())
	}
}

func TestUpdateModelAndFilter(t *testing.T) {
	update := NewPgUpdate(testModel{relation: "users"})

	if update.Model().Relation() != "users" {
		t.Fatalf("expected relation users, got %q", update.Model().Relation())
	}
	if update.Filter() == nil {
		t.Fatal("expected non-nil filter")
	}
}
