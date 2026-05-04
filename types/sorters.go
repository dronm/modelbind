package types

type DBSorters interface {
	Add(fieldID string, direct SQLSortDirect)
	SQL() string
	Len() int
}
