package pg

import (
	"testing"
)

func TestLimitSQL(t *testing.T) {

	tests := []struct {
		limit  PgLimit
		expSql string
	}{
		{PgLimit{from: 1, count: 10}, " OFFSET 1 LIMIT 10"},
		{PgLimit{count: 25}, " LIMIT 25"},
	}
	for _, test := range tests {
		t.Run(t.Name(), func(t *testing.T) {
			gotSql := test.limit.SQL()
			if test.expSql != gotSql {
				t.Fatalf("expected %s, got %s", test.expSql, gotSql)
			}
		})
	}
}
