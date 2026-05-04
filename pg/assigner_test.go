package pg

import (
	"reflect"
	"testing"
)

func TestAssignerSQL(t *testing.T) {
	params := []any{"existing"}
	assigner := PgAssigner{fieldID: "name", value: "Alice"}

	gotSQL := assigner.SQL(&params)
	wantSQL := "name = $2"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{"existing", "Alice"}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}
}

func TestAssignerSQLWithQualifiedField(t *testing.T) {
	params := []any{}
	assigner := PgAssigner{fieldID: "u.name", value: "Alice"}

	gotSQL := assigner.SQL(&params)
	wantSQL := "u.name = $1"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}
}

func TestAssignersSQL(t *testing.T) {
	params := []any{}
	assigners := PgAssigners{}
	assigners.Add("name", "Alice")
	assigners.Add("age", 30)

	gotSQL := assigners.SQL(&params)
	wantSQL := "name = $1, age = $2"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{"Alice", 30}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}

	if assigners.Len() != 2 {
		t.Fatalf("expected len 2, got %d", assigners.Len())
	}
}

func TestAssignersSQLEmpty(t *testing.T) {
	params := []any{}
	assigners := PgAssigners{}

	gotSQL := assigners.SQL(&params)
	if gotSQL != "" {
		t.Fatalf("expected empty SQL, got %q", gotSQL)
	}
	if len(params) != 0 {
		t.Fatalf("expected no params, got %#v", params)
	}
}

func TestAssignerSQLPanicsOnUnsafeField(t *testing.T) {
	params := []any{}
	assigner := PgAssigner{fieldID: "name;DROP TABLE users", value: "Alice"}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unsafe field reference")
		}
	}()

	_ = assigner.SQL(&params)
}
