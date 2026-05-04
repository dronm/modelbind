package pg

import "testing"

func TestLimitSQLZeroCount(t *testing.T) {
	limit := NewPgLimit(10, 0)

	gotSQL := limit.SQL()
	if gotSQL != "" {
		t.Fatalf("expected empty SQL, got %q", gotSQL)
	}
}

func TestLimitAccessorsAndSetters(t *testing.T) {
	limit := NewPgLimit(0, 10)

	if limit.From() != 0 {
		t.Fatalf("expected from 0, got %d", limit.From())
	}
	if limit.Count() != 10 {
		t.Fatalf("expected count 10, got %d", limit.Count())
	}

	limit.SetFrom(20)
	limit.SetCount(30)

	if limit.From() != 20 {
		t.Fatalf("expected from 20, got %d", limit.From())
	}
	if limit.Count() != 30 {
		t.Fatalf("expected count 30, got %d", limit.Count())
	}
	if limit.SQL() != " OFFSET 20 LIMIT 30" {
		t.Fatalf("expected offset limit SQL, got %q", limit.SQL())
	}
}
