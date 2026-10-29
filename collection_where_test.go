package modelbind

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/pg"
	"github.com/dronm/modelbind/types"
)

type whereOrder struct {
	ID         int       `json:"id"`
	CustomerID int       `json:"customer_id"`
	OrderDate  time.Time `json:"order_date" dateType:"date"`
	Name       string    `json:"name" max:"20"`
}

func (*whereOrder) Relation() string { return "orders" }
func (*whereOrder) CollectionAgg() any {
	return &struct {
		Count int `json:"count" agg:"count(*)"`
	}{}
}

const cursorWhere = `{"and":[{"f":"customer_id","o":"e","v":10},{"or":[{"f":"order_date","o":"l","v":"2026-10-29"},{"and":[{"f":"order_date","o":"e","v":"2026-10-29"},{"f":"id","o":"l","v":1500}]}]}]}`

func TestCollectionWhereCursorAndTotals(t *testing.T) {
	query := pg.NewPgGroupedSelect(&whereOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
	params := CollectionParamsWithWhere{
		CollectionParams: CollectionParams{Sorter: []CollectionSorter{{"order_date", SortParDesc}, {"id", SortParDesc}}, Count: 100},
		Where:            json.RawMessage(cursorWhere),
	}
	if err := BindCollectionSelectModelWithWhere(query, params); err != nil {
		t.Fatal(err)
	}
	args := []any{}
	data, totals, err := query.BuildCollectionSQL(&args)
	if err != nil {
		t.Fatal(err)
	}
	wantWhere := " WHERE ((customer_id = $1) AND ((order_date < $2) OR ((order_date = $3) AND (id < $4))))"
	if data != "SELECT id,customer_id,order_date,name FROM orders"+wantWhere+" ORDER BY order_date DESC, id DESC LIMIT 100" {
		t.Fatal(data)
	}
	if totals != "SELECT count(*) FROM orders"+wantWhere {
		t.Fatal(totals)
	}
	if !reflect.DeepEqual(args, []any{float64(10), "2026-10-29", "2026-10-29", float64(1500)}) {
		t.Fatal(args)
	}
}

func TestCollectionWhereComposesLegacyRequestAndPolicy(t *testing.T) {
	query := pg.NewPgGroupedSelect(&whereOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
	legacy := CollectionParams{Filter: []CollectionFilter{
		{Fields: map[string]CollectionFilterField{"id": {FilterOperParEq, 1}}},
		{Join: FilterParJoinOr, Fields: map[string]CollectionFilterField{"id": {FilterOperParEq, 2}}},
		{Fields: map[string]CollectionFilterField{"customer_id": {FilterOperParEq, 10}}},
	}}
	if err := query.SetAccessPolicy(access.Scope(access.Eq("owner", 42)), pg.PolicyBinding{Rows: pg.FieldMap{"owner": "customer_id"}}); err != nil {
		t.Fatal(err)
	}
	if err := query.SetRequestPredicate(access.Greater("id", 0), pg.FieldMap{"id": "id"}); err != nil {
		t.Fatal(err)
	}
	if err := BindCollectionSelectModelWithWhere(query, CollectionParamsWithWhere{legacy, json.RawMessage(`{"f":"id","o":"l","v":1500}`)}); err != nil {
		t.Fatal(err)
	}
	args := []any{"existing"}
	data, totals, err := query.BuildCollectionSQL(&args)
	if err != nil {
		t.Fatal(err)
	}
	want := " WHERE (((id = $2) OR (id = $3) AND (customer_id = $4)) AND (id > $5) AND (customer_id = $6)) AND (id < $7)"
	for _, sql := range []string{data, totals} {
		if !strings.Contains(sql, want) {
			t.Fatal(sql)
		}
	}
	if !reflect.DeepEqual(args, []any{"existing", 1, 2, 10, int64(0), int64(42), float64(1500)}) {
		t.Fatal(args)
	}
}

func TestCollectionWhereAllOperatorsKeepLegacySemantics(t *testing.T) {
	cases := []struct {
		op    FilterOperatorParam
		value any
		want  string
	}{
		{FilterOperParEq, 2, "id = $2"}, {FilterOperParLess, 2, "id < $2"},
		{FilterOperParGr, 2, "id > $2"}, {FilterOperParLessEq, 2, "id <= $2"},
		{FilterOperParGrEq, 2, "id >= $2"}, {FilterOperParNotEq, 2, "id <> $2"},
		{FilterOperParLk, 2, "id LIKE $2"}, {FilterOperParILk, 2, "id ILIKE $2"},
		{FilterOperParIs, nil, "id IS NULL"}, {FilterOperParIn, nil, "id IS NOT NULL"},
		{FilterOperParEq, nil, "id = $2"}, {FilterOperParNotEq, nil, "id <> $2"},
		{FilterOperParIncl, []int{1, 2}, "id = ANY($2)"},
		{FilterOperParAny, []int{1, 2}, "$2 = ANY(id)"},
		{FilterOperParHas, 2, "$2 = ANY(id)"},
		{FilterOperParOverlap, []int{1, 2}, "id && $2"},
		{FilterOperParContains, []int{1, 2}, "id @> $2"},
		{FilterOperParTS, 2, "id @@ to_tsquery('russian', $2)"},
	}
	for _, tc := range cases {
		t.Run(string(tc.op)+tc.want, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"and": []any{map[string]any{"f": "id", "o": tc.op, "v": tc.value}}})
			if err != nil {
				t.Fatal(err)
			}
			query := pg.NewPgGroupedSelect(&whereOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
			if err := BindCollectionSelectModelWithWhere(query, CollectionParamsWithWhere{Where: raw}); err != nil {
				t.Fatal(err)
			}
			args := []any{"prefix"}
			sql, totals, err := query.BuildCollectionSQL(&args)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(sql, " WHERE (("+tc.want+"))") || !strings.Contains(totals, " WHERE (("+tc.want+"))") {
				t.Fatalf("%s; %s", sql, totals)
			}
			var legacyValue any
			valueJSON, _ := json.Marshal(tc.value)
			if err := json.Unmarshal(valueJSON, &legacyValue); err != nil {
				t.Fatal(err)
			}
			filters := pg.PgFilters{}
			if err := ParseFilterParams(&whereOrder{}, &filters, CollectionParams{Filter: []CollectionFilter{{Fields: map[string]CollectionFilterField{"id": {tc.op, legacyValue}}}}}, ""); err != nil {
				t.Fatal(err)
			}
			legacyArgs := []any{"prefix"}
			filters.SQL(&legacyArgs)
			if !reflect.DeepEqual(args, legacyArgs) {
				t.Fatalf("%#v != %#v", args, legacyArgs)
			}
		})
	}
}

func TestCollectionWhereRejectsInvalidExpressions(t *testing.T) {
	cases := []struct{ raw, path string }{
		{`null`, "where"}, {`[]`, "where"}, {`{}`, "where"}, {`true`, "where"},
		{`{"and":[]}`, "where"}, {`{"or":null}`, "where"}, {`{"and":{}}`, "where"},
		{`{"and":[{}]}`, "where.and[0]"}, {`{"or":[{"and":[]}]}`, "where.or[0]"},
		{`{"and":[],"or":[]}`, "where"}, {`{"f":"id","o":"e"}`, "where"},
		{`{"f":"id","o":"e","v":1,"sql":"TRUE"}`, "where"},
		{`{"f":"id","o":"bad","v":1}`, "where"}, {`{"f":"id","o":null,"v":1}`, "where"},
		{`{"f":"id","f":"name","o":"e","v":1}`, "where"},
		{`{"f":"missing","o":"e","v":1}`, "where"},
		{`{"f":"id;DROP TABLE orders","o":"e","v":1}`, "where"},
		{`{"or":[{"f":"name","o":"e","v":123}]}`, "where.or[0]"},
		{`{"f":"name","o":"e","v":"abcdefghijklmnopqrstuvwxyz"}`, "where"},
		{`{"f":"order_date","o":"l","v":"wrong"}`, "where"},
		{`{"f":"id","o":"e","v":1} {}`, "where"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			query := pg.NewPgGroupedSelect(&whereOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
			err := BindCollectionSelectModelWithWhere(query, CollectionParamsWithWhere{Where: json.RawMessage(tc.raw)})
			if err == nil || !strings.Contains(err.Error(), tc.path) {
				t.Fatalf("%v", err)
			}
			args := []any{"preserved"}
			if _, _, err := query.BuildCollectionSQL(&args); err == nil {
				t.Fatal("invalid bind must fail closed")
			}
			if !reflect.DeepEqual(args, []any{"preserved"}) {
				t.Fatal(args)
			}
		})
	}
}

func TestCollectionWhereNestingLimit(t *testing.T) {
	raw := `{"f":"id","o":"e","v":1}`
	for i := 0; i < MaxCollectionWhereDepth; i++ {
		raw = `{"and":[` + raw + `]}`
	}
	if _, err := ParseCollectionWhere(&whereOrder{}, json.RawMessage(raw), ""); err != nil {
		t.Fatal(err)
	}
	raw = `{"or":[` + raw + `]}`
	if err := ValidateCollectionWhere(json.RawMessage(raw)); err == nil || !strings.Contains(err.Error(), "nesting") {
		t.Fatal(err)
	}
}

func TestCollectionWhereParametersAndFieldValidation(t *testing.T) {
	value := `x' OR TRUE --`
	raw, _ := json.Marshal(map[string]any{"f": "name", "o": "e", "v": value})
	query := pg.NewPgGroupedSelect(&whereOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
	if err := BindCollectionSelectModelWithWhere(query, CollectionParamsWithWhere{Where: raw}); err != nil {
		t.Fatal(err)
	}
	args := []any{}
	sql, err := query.BuildSQL(&args)
	if err != nil || strings.Contains(sql, value) || !reflect.DeepEqual(args, []any{value}) {
		t.Fatalf("%s %#v %v", sql, args, err)
	}
	if _, err := ParseCollectionWhere(&whereOrder{}, raw, "unsafe;table"); err == nil {
		t.Fatal("unsafe table accepted")
	}
}

// These literals and implementations intentionally exercise the original API.
// No new methods may be required of third-party builders.
var _ = CollectionParams{nil, nil, 0, 20}
var _ = CollectionFilter{"or", map[string]CollectionFilterField{"id": {"e", 1}}}
var _ = CollectionSorter{"id", "d"}
var _ types.DBFilters = (*compatWhereFilters)(nil)
var _ types.DBSelecter = (*compatWhereSelect)(nil)

type compatWhereFilters struct{}

func (*compatWhereFilters) Add(string, string, any, types.SQLFilterOperator, types.FilterJoin) {}
func (*compatWhereFilters) AddFullTextSearch(string, string, any, types.FilterJoin)            {}
func (*compatWhereFilters) AddArrayInclude(string, string, any, types.FilterJoin)              {}
func (*compatWhereFilters) AddColumnArrayInclude(string, string, any, types.FilterJoin)        {}
func (*compatWhereFilters) Len() int                                                           { return 0 }

type compatWhereSelect struct{}

func (*compatWhereSelect) Model() types.DBAggModel         { return &whereOrder{} }
func (*compatWhereSelect) Filter() types.DBFilters         { return &compatWhereFilters{} }
func (*compatWhereSelect) SetFilter(types.DBFilters) error { return nil }
func (*compatWhereSelect) Limit() types.DBLimit            { return &pg.PgLimit{} }
func (*compatWhereSelect) Sorter() types.DBSorters         { return &pg.PgSorters{} }
func (*compatWhereSelect) AddField(string, any)            {}
func (*compatWhereSelect) AddAggField(string, any)         {}

func TestLegacyCollectionBindingUnchanged(t *testing.T) {
	if err := BindCollectionSelectModel(&compatWhereSelect{}, CollectionParams{}); err != nil {
		t.Fatal(err)
	}
	var params CollectionParams
	raw := `{"filter":[{"f":{"id":{"o":"unknown","v":1,"ignored":true}},"j":"unexpected","ignored":true},{"f":{"id":{"o":"e","v":2}},"j":"or"},{"f":{"customer_id":{"o":"e","v":10}}}],"where":{"malformed":true},"extra":true,"count":5}`
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatal(err)
	}
	legacy := pg.NewPgSelect(&whereOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
	grouped := pg.NewPgGroupedSelect(&whereOrder{}, &pg.PgFilters{}, &pg.PgSorters{}, &pg.PgLimit{})
	if err := BindCollectionSelectModel(legacy, params); err != nil {
		t.Fatal(err)
	}
	if err := BindCollectionSelectModelWithWhere(grouped, CollectionParamsWithWhere{CollectionParams: params}); err != nil {
		t.Fatal(err)
	}
	args, extendedArgs := []any{}, []any{}
	sql, totals, err := legacy.BuildCollectionSQL(&args)
	extendedSQL, extendedTotals, extendedErr := grouped.BuildCollectionSQL(&extendedArgs)
	if err != nil || extendedErr != nil || sql != extendedSQL || totals != extendedTotals || !reflect.DeepEqual(args, extendedArgs) {
		t.Fatalf("%s %s %v %v", sql, extendedSQL, err, extendedErr)
	}
	if !strings.Contains(sql, " WHERE (id = $1) OR (id = $2) AND (customer_id = $3)") {
		t.Fatal(sql)
	}
}
