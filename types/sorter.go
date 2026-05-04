package types

type SQLSortDirect string

const (
	SQLSortAsc  SQLSortDirect = "ASC"
	SQLSortDesc SQLSortDirect = "DESC"
)

type SQLSorter interface {
	FieldID() string
	Direct() SQLSortDirect
}
