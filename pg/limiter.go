package pg

import (
	"fmt"
)

type PgLimit struct {
	from  int
	count int
}

func NewPgLimit(from, count int) *PgLimit {
	return &PgLimit{from: from, count: count}
}

func (l PgLimit) From() int {
	return l.from
}

func (l *PgLimit) SetFrom(v int) {
	l.from = v
}

func (l PgLimit) Count() int {
	return l.count
}

func (l *PgLimit) SetCount(v int) {
	l.count = v
}

func (e PgLimit) SQL() string {
	if e.count == 0 {
		return ""
	}

	if e.from == 0 {
		return fmt.Sprintf(" LIMIT %d", e.count)
	}

	return fmt.Sprintf(" OFFSET %d LIMIT %d", e.from, e.count)
}
