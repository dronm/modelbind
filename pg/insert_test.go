package pg

import (
	"reflect"
	"testing"
)

func TestInsertSQL(t *testing.T) {
	params := []any{"existing"}
	insert := NewPgInsert(testModel{relation: "users"})
	insert.AddField("name", "Alice")
	insert.AddField("age", 30)

	gotSQL := insert.SQL(&params)
	wantSQL := "INSERT INTO users (name,age) VALUES ($2,$3)"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantParams := []any{"existing", "Alice", 30}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("expected params %#v, got %#v", wantParams, params)
	}

	if insert.InsertFieldLen() != 2 {
		t.Fatalf("expected field len 2, got %d", insert.InsertFieldLen())
	}
}

func TestInsertSQLReturning(t *testing.T) {
	params := []any{}
	insert := NewPgInsert(testModel{relation: "users"})
	insert.AddField("name", "Alice")
	insert.AddRetField("id", nil)
	insert.AddRetField("created_at", nil)

	gotSQL := insert.SQL(&params)
	wantSQL := "INSERT INTO users (name) VALUES ($1) RETURNING id,created_at"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	wantRetFields := []string{"id", "created_at"}
	if !reflect.DeepEqual(insert.RetFieldIds(), wantRetFields) {
		t.Fatalf("expected return field ids %#v, got %#v", wantRetFields, insert.RetFieldIds())
	}

	wantRetMap := map[string]any{"id": nil, "created_at": nil}
	if !reflect.DeepEqual(insert.RetFields(), wantRetMap) {
		t.Fatalf("expected return fields %#v, got %#v", wantRetMap, insert.RetFields())
	}
}

func TestInsertModel(t *testing.T) {
	insert := NewPgInsert(testModel{relation: "users"})

	if insert.Model().Relation() != "users" {
		t.Fatalf("expected relation users, got %q", insert.Model().Relation())
	}
}

func TestInsertSQLPanicsOnUnsafeField(t *testing.T) {
	params := []any{}
	insert := NewPgInsert(testModel{relation: "users"})
	insert.AddField("name);DROP TABLE users", "Alice")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unsafe field reference")
		}
	}()

	_ = insert.SQL(&params)
}
