package pg

import (
	"testing"

	"github.com/dronm/modelbind/types"
)

func TestSorterSQLWithFieldPrefix(t *testing.T) {
	sorter := PgSorter{fieldID: "name", direct: types.SQLSortAsc, fieldPref: "u"}

	gotSQL := sorter.SQL()
	wantSQL := "u.name ASC"
	if gotSQL != wantSQL {
		t.Fatalf("expected SQL %q, got %q", wantSQL, gotSQL)
	}

	if sorter.FieldID() != "name" {
		t.Fatalf("expected field id name, got %q", sorter.FieldID())
	}
	if sorter.Direct() != types.SQLSortAsc {
		t.Fatalf("expected direction ASC, got %q", sorter.Direct())
	}
	if sorter.FieldPref() != "u" {
		t.Fatalf("expected field prefix u, got %q", sorter.FieldPref())
	}
}

func TestSortersAddLenAndEmptySQL(t *testing.T) {
	var sorters PgSorters

	if sorters.SQL() != "" {
		t.Fatalf("expected empty SQL, got %q", sorters.SQL())
	}
	if sorters.Len() != 0 {
		t.Fatalf("expected len 0, got %d", sorters.Len())
	}

	sorters.Add("name", types.SQLSortDesc)
	if sorters.Len() != 1 {
		t.Fatalf("expected len 1, got %d", sorters.Len())
	}
	if sorters.SQL() != " ORDER BY name DESC" {
		t.Fatalf("expected order by SQL, got %q", sorters.SQL())
	}
}

func TestSorterSQLPanicsOnUnsafeField(t *testing.T) {
	sorter := PgSorter{fieldID: "name;DROP TABLE users", direct: types.SQLSortAsc}

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unsafe field reference")
		}
	}()

	_ = sorter.SQL()
}
