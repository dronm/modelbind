package pg

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/types"
)

// Keep representative positional literals buildable inside the adapter too.
var _ = PgFilter{"id", 1, types.SQLFilterOperatorEq, "", types.SQLFilterJoinAnd, ""}
var _ = PgSelect{nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}

func TestGroupedSelectBuildFailuresAreTransactional(t *testing.T) {
	filters := PgFilters{}
	filters.Add("", "id", 1, types.SQLFilterOperatorEq, types.SQLFilterJoinAnd)
	query := NewPgGroupedSelect(testModel{relation: "orders"}, &PgFilters{}, nil, nil)
	query.AddField("id", nil)
	query.AddAggField("count(*)", nil)
	if err := query.SetWhere(FilterAnd(FilterLeaf(filters))); err != nil {
		t.Fatal(err)
	}
	if _, _, err := query.BuildCollectionSQL(nil); err == nil {
		t.Fatal("nil parameter pointer")
	}
	if err := query.SetAccessPolicy(access.DenyAll(), PolicyBinding{}); !errors.Is(err, access.ErrDenied) {
		t.Fatal(err)
	}
	args := []any{"existing"}
	data, totals, err := query.BuildCollectionSQL(&args)
	if !errors.Is(err, access.ErrDenied) || data != "" || totals != "" || !reflect.DeepEqual(args, []any{"existing"}) {
		t.Fatalf("%q %q %#v %v", data, totals, args, err)
	}
}

func TestGroupedSelectInvalidTreesFailClosed(t *testing.T) {
	cases := []FilterExpression{FilterExpression{}, FilterAnd(), FilterOr(), FilterLeaf(PgFilters{}), FilterAnd(FilterExpression{})}
	for _, expression := range cases {
		query := NewPgGroupedSelect(testModel{relation: "orders"}, nil, nil, nil)
		if err := query.SetWhere(expression); err == nil {
			t.Fatal("invalid tree accepted")
		}
		args := []any{1}
		if _, err := query.BuildSQL(&args); err == nil {
			t.Fatal("ignored setter error")
		}
		if !reflect.DeepEqual(args, []any{1}) {
			t.Fatal(args)
		}
	}
	query := PgGroupedSelect{}
	args := []any{}
	if _, err := query.BuildSQL(&args); err == nil {
		t.Fatal("uninitialized select")
	}
}

func TestGroupedSelectSingleQueryAndCollectionUseSameExpression(t *testing.T) {
	filters := PgFilters{}
	filters.Add("", "id", 5, types.SQLFilterOperatorLess, types.SQLFilterJoinAnd)
	query := NewPgGroupedSelect(testModel{relation: "orders"}, nil, nil, nil)
	query.AddField("id", nil)
	if err := query.SetWhere(FilterOr(FilterLeaf(filters))); err != nil {
		t.Fatal(err)
	}
	a, b := []any{}, []any{}
	single := query.SQL(&a)
	data, totals := query.CollectionSQL(&b)
	if single != data || totals != "" || !reflect.DeepEqual(a, b) || !strings.Contains(data, "WHERE ((id < $1))") {
		t.Fatalf("%s %s %s %#v %#v", single, data, totals, a, b)
	}
}
