package modelbind_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dronm/modelbind"
	"github.com/dronm/modelbind/pg"
)

type groupedExampleOrder struct {
	ID        int       `json:"id"`
	OrderDate time.Time `json:"order_date" dateType:"date"`
}

func (*groupedExampleOrder) Relation() string   { return "orders" }
func (*groupedExampleOrder) CollectionAgg() any { return nil }

func ExampleBindCollectionSelectModelWithWhere() {
	var request modelbind.CollectionParamsWithWhere
	err := json.Unmarshal([]byte(`{
		"where":{"or":[
			{"f":"order_date","o":"l","v":"2026-10-29"},
			{"and":[{"f":"order_date","o":"e","v":"2026-10-29"},{"f":"id","o":"l","v":1500}]}
		]},
		"sorter":[{"f":"order_date","d":"d"},{"f":"id","d":"d"}],
		"count":100
	}`), &request)
	if err != nil {
		panic(err)
	}
	query := pg.NewPgGroupedSelect(&groupedExampleOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
	if err := modelbind.BindCollectionSelectModelWithWhere(query, request); err != nil {
		panic(err)
	}
	args := []any{}
	sql, err := query.BuildSQL(&args)
	if err != nil {
		panic(err)
	}
	fmt.Println(sql)
	fmt.Println(args)
	// Output:
	// SELECT id,order_date FROM orders WHERE ((order_date < $1) OR ((order_date = $2) AND (id < $3))) ORDER BY order_date DESC, id DESC LIMIT 100
	// [2026-10-29 2026-10-29 1500]
}
