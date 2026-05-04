package types

type DBLimit interface {
	From() int
	SetFrom(int)
	Count() int
	SetCount(int)
	SQL() string
}
