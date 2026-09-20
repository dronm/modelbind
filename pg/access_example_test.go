package pg_test

import (
	"fmt"

	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/pg"
	"github.com/dronm/modelbind/types"
)

type exampleOrder struct{}

func (exampleOrder) Relation() string { return "orders" }

func ExamplePgUpdate_SetAccessPolicy() {
	query := pg.NewPgUpdate(exampleOrder{})
	query.AddField("name", "updated")
	query.Filter().Add("", "id", 100, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)

	policy := access.Policy{
		Decision: access.Restricted,
		Rows:     access.Eq("customer_id", 42),
		Writes:   access.WriteRules{"customer_id": access.Enforce(42)},
	}
	if err := query.SetAccessPolicy(policy, pg.PolicyBinding{
		Rows:   pg.FieldMap{"customer_id": "customer_id"},
		Writes: pg.FieldMap{"customer_id": "customer_id"},
	}); err != nil {
		panic(err)
	}
	params := []any{}
	sql, err := query.BuildSQL(&params)
	if err != nil {
		panic(err)
	}
	fmt.Println(sql)
	fmt.Println(params)
	// Output:
	// UPDATE orders SET name = $1 WHERE (id = $2) AND (customer_id = $3)
	// [updated 100 42]
}

func ExampleCompilePredicate() {
	predicate := access.And(
		access.Or(access.Eq("status", "draft"), access.Eq("status", "new")),
		access.In("customer_id", 42, 43),
	)
	params := []any{"previous parameter"}
	sql, err := pg.CompilePredicate(predicate, pg.FieldMap{
		"status":      "o.status",
		"customer_id": "o.customer_id",
	}, &params)
	if err != nil {
		panic(err)
	}
	fmt.Println(sql)
	fmt.Println(params)
	// Output:
	// ((o.status = $2) OR (o.status = $3)) AND (o.customer_id IN ($4,$5))
	// [previous parameter draft new 42 43]
}
