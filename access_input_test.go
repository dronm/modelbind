package modelbind

import (
	"errors"
	"math"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/pg"
	"github.com/dronm/modelbind/types"
)

type policyStatus string

type policyInput struct {
	ID         *int          `json:"id,omitempty" srvCalc:""`
	Name       *string       `json:"name,omitempty" required:""`
	CustomerID *int          `json:"customer_id,omitempty" required:""`
	SiteID     *int          `json:"site_id,omitempty"`
	Status     *policyStatus `json:"status,omitempty" valList:"draft,posted"`
}

func (*policyInput) Relation() string       { return "orders" }
func (*policyInput) AccessResource() string { return "Order" }

type policyKey struct {
	ID *int `json:"id"`
}

func decodePolicyInput(t *testing.T, body string) ModelInput[*policyInput] {
	t.Helper()
	r := httptest.NewRequest("POST", "/orders", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	input, err := DecodeJSONInput[*policyInput](r)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func policyInputBinding() pg.PolicyBinding {
	return pg.PolicyBinding{
		Rows:   pg.FieldMap{"customer_id": "customer_id"},
		Writes: pg.FieldMap{"customer_id": "customer_id", "status": "status"},
	}
}

func TestPrepareInputBeforeRequiredValidation(t *testing.T) {
	input := decodePolicyInput(t, `{"name":"order"}`)
	if err := input.Validate(true); err == nil {
		t.Fatal("original input should lack required customer_id")
	}
	rules := access.WriteRules{"customer_id": access.Enforce(int64(42)), "status": access.Default("draft")}
	effective, err := PrepareModelInput(input, access.Insert, rules)
	if err != nil {
		t.Fatal(err)
	}
	if err := effective.Validate(true); err != nil {
		t.Fatal(err)
	}
	if effective.Model.CustomerID == nil || *effective.Model.CustomerID != 42 || effective.IsAbsent("customer_id") {
		t.Fatal("server field not supplied/marked present")
	}
	if effective.Model.Status == nil || *effective.Model.Status != policyStatus("draft") || effective.IsAbsent("status") {
		t.Fatal("named string enum default not applied")
	}
	if effective.Model == input.Model || input.Model.CustomerID != nil || input.Model.Status != nil || !input.IsAbsent("customer_id") || !input.IsAbsent("status") {
		t.Fatal("original request was mutated")
	}
	*effective.Model.CustomerID = 99
	if input.Model.CustomerID != nil {
		t.Fatal("server pointer aliases request")
	}
}

func TestPrepareInputRejectsConflictsAndExplicitNull(t *testing.T) {
	for _, body := range []string{`{"name":"order","customer_id":99}`, `{"name":"order","customer_id":null}`} {
		input := decodePolicyInput(t, body)
		_, err := PrepareModelInput(input, access.Insert, access.WriteRules{"customer_id": access.Enforce(42)})
		if !errors.Is(err, access.ErrWriteConflict) {
			t.Fatalf("conflict accepted for %s: %v", body, err)
		}
	}
	input := decodePolicyInput(t, `{"name":"order","customer_id":42,"status":null}`)
	effective, err := PrepareModelInput(input, access.Insert, access.WriteRules{"status": access.Default("draft")})
	if err != nil || effective.Model.Status != nil || effective.IsAbsent("status") {
		t.Fatalf("default overwrote explicit NULL: %#v %v", effective, err)
	}
	// Untracked zero/nil values are present; defaults must not infer absence.
	input = ModelInput[*policyInput]{Model: &policyInput{}}
	if _, err := PrepareModelInput(input, access.Insert, access.WriteRules{"customer_id": access.Enforce(42)}); !errors.Is(err, access.ErrWriteConflict) {
		t.Fatalf("untracked nil treated as omitted field: %v", err)
	}
}

func TestPrepareUpdateDoesNotWriteAbsentEnforcedField(t *testing.T) {
	input := decodePolicyInput(t, `{"name":"updated"}`)
	effective, err := PrepareModelInput(input, access.Update, access.WriteRules{"customer_id": access.Enforce(42)})
	if err != nil || effective.Model.CustomerID != nil || !effective.IsAbsent("customer_id") {
		t.Fatalf("update synthesized an absent assignment: %#v %v", effective, err)
	}
	input = decodePolicyInput(t, `{"name":"updated","customer_id":42}`)
	effective, err = PrepareModelInput(input, access.Update, access.WriteRules{"customer_id": access.Enforce(42)})
	if err != nil {
		t.Fatal(err)
	}
	if effective.Model.CustomerID == input.Model.CustomerID {
		t.Fatal("governed pointer was not copied")
	}
	*effective.Model.CustomerID = 99
	if *input.Model.CustomerID != 42 {
		t.Fatal("original field was mutated")
	}
}

func TestPrepareInputRejectsUnknownAndDatabaseGeneratedFields(t *testing.T) {
	input := decodePolicyInput(t, `{"name":"order"}`)
	for _, field := range []string{"missing_owner", "id"} {
		_, err := PrepareModelInput(input, access.Insert, access.WriteRules{field: access.Enforce(42)})
		if !errors.Is(err, access.ErrInvalidField) {
			t.Fatalf("field %q accepted: %v", field, err)
		}
	}
	if _, err := PrepareModelInput(ModelInput[*policyInput]{}, access.Insert, nil); err == nil {
		t.Fatal("nil model accepted")
	}
	duplicateType := reflect.StructOf([]reflect.StructField{
		{Name: "A", Type: reflect.TypeOf((*int)(nil)), Tag: `json:"id"`},
		{Name: "B", Type: reflect.TypeOf((*int)(nil)), Tag: `json:"id"`},
	})
	duplicate := reflect.New(duplicateType).Interface()
	if _, err := PrepareModelInput(ModelInput[any]{Model: duplicate}, access.Insert, nil); !errors.Is(err, access.ErrInvalidField) {
		t.Fatal("duplicate input tags accepted")
	}
}

func TestPolicyFieldValueConversionsAreLossless(t *testing.T) {
	type namedInt int32
	type namedString string
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		value  any
		target reflect.Type
		valid  bool
	}{
		{int64(42), reflect.TypeOf(int8(0)), true},
		{int64(128), reflect.TypeOf(int8(0)), false},
		{uint64(math.MaxUint64), reflect.TypeOf(int64(0)), false},
		{int64(-1), reflect.TypeOf(uint64(0)), false},
		{uint64(42), reflect.TypeOf(namedInt(0)), true},
		{int64(42), reflect.TypeOf(uint8(0)), true},
		{uint64(256), reflect.TypeOf(uint8(0)), false},
		{float64(42), reflect.TypeOf(int(0)), false},
		{int64(42), reflect.TypeOf(""), false},
		{"42", reflect.TypeOf(int(0)), false},
		{"draft", reflect.TypeOf(namedString("")), true},
		{float64(0.5), reflect.TypeOf(float32(0)), true},
		{float64(0.1), reflect.TypeOf(float32(0)), false},
		{float64(math.MaxFloat64), reflect.TypeOf(float32(0)), false},
		{true, reflect.TypeOf(false), true},
		{now, reflect.TypeOf((*time.Time)(nil)), true},
		{nil, reflect.TypeOf((*int)(nil)), true},
		{nil, reflect.TypeOf(int(0)), false},
		{int64(42), reflect.TypeOf((**int)(nil)), true},
	}
	for _, tt := range tests {
		_, err := policyFieldValue(tt.value, tt.target)
		if (err == nil) != tt.valid {
			t.Errorf("convert %T to %s: %v, valid %v", tt.value, tt.target, err, tt.valid)
		}
	}
}

func TestPolicyAwareInsertBindingAndReturning(t *testing.T) {
	input := decodePolicyInput(t, `{"name":"new order"}`)
	insert := pg.NewPgInsert(input.Model)
	policy := access.ForInsert(access.WriteRules{"customer_id": access.Enforce(42), "status": access.Default("draft")})
	if err := insert.SetAccessPolicy(policy, policyInputBinding()); err != nil {
		t.Fatal(err)
	}
	effective, err := BindInsertModelInputWithPolicy(input, insert)
	if err != nil {
		t.Fatal(err)
	}
	params := []any{}
	sql, err := insert.BuildSQL(&params)
	want := "INSERT INTO orders (name,customer_id,status) VALUES ($1,$2,$3) RETURNING id"
	if err != nil || sql != want || !reflect.DeepEqual(params, []any{"new order", int64(42), "draft"}) {
		t.Fatalf("bad policy-aware binding: %q %#v %v", sql, params, err)
	}
	if *effective.Model.CustomerID != 42 || input.Model.CustomerID != nil {
		t.Fatal("effective/original input confused")
	}
	// RETURNING targets must belong to the returned effective model.
	id := 123
	target, ok := insert.RetFieldValues()[0].(**int)
	if !ok {
		t.Fatalf("unexpected RETURNING target: %T", insert.RetFieldValues()[0])
	}
	*target = &id
	if effective.Model.ID == nil || *effective.Model.ID != 123 || input.Model.ID != nil {
		t.Fatal("RETURNING is not attached to effective model")
	}
}

func TestPolicyAwareUpdateBindingChecksKeysAndOwnership(t *testing.T) {
	input := decodePolicyInput(t, `{"name":"updated"}`)
	policy := access.Scope(access.Eq("customer_id", 42))
	policy.Writes = access.WriteRules{"customer_id": access.Enforce(42)}
	update := pg.NewPgUpdate(input.Model)
	if err := update.SetAccessPolicy(policy, policyInputBinding()); err != nil {
		t.Fatal(err)
	}
	if _, err := BindUpdateModelInputWithPolicy(&policyKey{}, input, update); !errors.Is(err, ErrMissingKey) {
		t.Fatalf("policy scope substituted for missing keys: %v", err)
	}
	if update.Filter().Len() != 0 || update.AssignerLen() != 0 {
		t.Fatal("failed key check partially bound update")
	}
	id := 10
	if _, err := BindUpdateModelInputWithPolicy(&policyKey{ID: &id}, input, update); err != nil {
		t.Fatal(err)
	}
	params := []any{}
	sql, err := update.BuildSQL(&params)
	if err != nil || sql != "UPDATE orders SET name = $1 WHERE (id = $2) AND (customer_id = $3)" || len(params) != 3 || params[2] != int64(42) {
		t.Fatalf("bad update: %q %#v %v", sql, params, err)
	}
}

func TestPolicyAwareBindRequiresInstalledPolicy(t *testing.T) {
	input := decodePolicyInput(t, `{"name":"order","customer_id":42}`)
	insert := pg.NewPgInsert(input.Model)
	if _, err := BindInsertModelInputWithPolicy(input, insert); !errors.Is(err, access.ErrInvalidPolicy) {
		t.Fatalf("missing policy accepted: %v", err)
	}
	if insert.InsertFieldLen() != 0 {
		t.Fatal("missing policy partially bound input")
	}
	_ = insert.SetAccessPolicy(access.DenyAll(), pg.PolicyBinding{})
	if _, err := BindInsertModelInputWithPolicy(input, insert); !errors.Is(err, access.ErrDenied) {
		t.Fatalf("denied policy accepted: %v", err)
	}
	var nilInsert *pg.PgInsert
	if _, err := BindInsertModelInputWithPolicy(input, nilInsert); !errors.Is(err, access.ErrInvalidPolicy) {
		t.Fatalf("typed nil builder accepted: %v", err)
	}
	if _, err := BindInsertModelInputWithPolicy(input, &noPolicyInserter{model: input.Model}); !errors.Is(err, access.ErrInvalidPolicy) {
		t.Fatalf("non-policy builder accepted: %v", err)
	}
}

type noPolicyInserter struct{ model types.DBModel }

func (b *noPolicyInserter) Model() types.DBModel  { return b.model }
func (*noPolicyInserter) AddField(string, any)    {}
func (*noPolicyInserter) AddRetField(string, any) {}

func TestRequireModelKeys(t *testing.T) {
	zero := 0
	var inner *int
	tests := []struct {
		key   any
		valid bool
	}{
		{nil, false},
		{(*policyKey)(nil), false},
		{&policyKey{}, false},
		{struct{}{}, false},
		{&policyKey{ID: &zero}, true},
		{struct {
			ID int `json:"id"`
		}{ID: 1}, true},
		{struct {
			ID   *int `json:"id"`
			Line *int `json:"line"`
		}{ID: &zero}, false},
		{struct {
			ID **int `json:"id"`
		}{ID: &inner}, false},
	}
	for _, tt := range tests {
		err := RequireModelKeys(tt.key)
		if (err == nil) != tt.valid {
			t.Errorf("keys %T: %v, valid %v", tt.key, err, tt.valid)
		}
		if err != nil && !errors.Is(err, ErrMissingKey) {
			t.Errorf("wrong error category: %v", err)
		}
	}
}

func TestPresenceCloneAndSetPresent(t *testing.T) {
	original := NewAbsentFieldSet()
	original.SetAbsent("owner")
	copy := original.Clone()
	copy.SetPresent("owner")
	copy.SetAbsent("other")
	if !original.IsAbsent("owner") || original.IsAbsent("other") || copy.IsAbsent("owner") {
		t.Fatal("presence copy is not independent")
	}
	var untracked AbsentFieldSet
	copy = untracked.Clone()
	copy.SetPresent("owner")
	if copy.IsTracked() {
		t.Fatal("untracked clone changed semantics")
	}
}
