package modelbind_test

import (
	"fmt"
	"net/http/httptest"
	"strings"

	"github.com/dronm/modelbind"
	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/pg"
)

type exampleOrderInput struct {
	ID         *int    `json:"id" srvCalc:""`
	Name       *string `json:"name" required:""`
	CustomerID *int    `json:"customer_id" required:""`
}

func (*exampleOrderInput) Relation() string { return "orders" }

func ExampleBindInsertModelInputWithPolicy() {
	request := httptest.NewRequest("POST", "/orders", strings.NewReader(`{"name":"new order"}`))
	request.Header.Set("Content-Type", "application/json")
	input, err := modelbind.DecodeJSONInput[*exampleOrderInput](request)
	if err != nil {
		panic(err)
	}

	// In the application, resolve this from trusted authenticated identity data.
	customerID := 42
	insert := pg.NewPgInsert(input.Model)
	if err := insert.SetAccessPolicy(access.ForInsert(access.WriteRules{
		"customer_id": access.Enforce(customerID),
	}), pg.PolicyBinding{
		Writes: pg.FieldMap{"customer_id": "customer_id"},
	}); err != nil {
		panic(err)
	}
	effective, err := modelbind.BindInsertModelInputWithPolicy(input, insert)
	if err != nil {
		panic(err)
	}
	params := []any{}
	query, err := insert.BuildSQL(&params)
	if err != nil {
		panic(err)
	}
	fmt.Println(query)
	fmt.Println(params)
	fmt.Println(*effective.Model.CustomerID, input.IsAbsent("customer_id"))
	// Output:
	// INSERT INTO orders (name,customer_id) VALUES ($1,$2) RETURNING id
	// [new order 42]
	// 42 true
}
