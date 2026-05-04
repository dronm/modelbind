package pg

import (
	"testing"

	"github.com/dronm/modelbind/types"
)

func TestSorterSQL(t *testing.T) {

	tests := []struct {
		sorter PgSorter
		expSQL string
	}{
		{PgSorter{fieldID: "f1", direct: types.SQLSortAsc}, "f1 ASC"},
		{PgSorter{fieldID: "f1", direct: types.SQLSortDesc}, "f1 DESC"},
	}
	for _, test := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			gotSql := test.sorter.SQL()
			if test.expSQL != gotSql {
				t.Fatalf("expected %s, got %s", test.expSQL, gotSql)
			}
		})
	}
}

func TestSortersSQL(t *testing.T) {
	tests := []struct {
		sorters PgSorters
		expSql  string
	}{
		{[]PgSorter{{fieldID: "f1", direct: types.SQLSortAsc}},
			" ORDER BY f1 ASC",
		},

		{[]PgSorter{{fieldID: "f1", direct: types.SQLSortAsc},
			{fieldID: "f2", direct: types.SQLSortDesc},
		},
			" ORDER BY f1 ASC, f2 DESC",
		},
	}
	for _, test := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			gotSQL := test.sorters.SQL()
			if test.expSql != gotSQL {
				t.Fatalf("expected %s, got %s", test.expSql, gotSQL)
			}
		})
	}
}
