package modelbind

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dronm/modelbind/pg"
)

type testInputModel struct {
	ID        *int       `json:"id,omitempty" primaryKey:""`
	Name      *string    `json:"name,omitempty" alias:"Name"`
	Age       *int       `json:"age,omitempty" alias:"Age" required:"" min:"18"`
	Active    *bool      `json:"active,omitempty"`
	BirthDate *time.Time `json:"birth_date,omitempty" dateType:"date"`
	Ignored   *string    `json:"-,omitempty"`
}

func (m *testInputModel) Relation() string {
	return "users"
}

type testInputKeyModel struct {
	ID *int `json:"id,omitempty"`
}

func (m *testInputKeyModel) Relation() string {
	return "users"
}

func TestDecodeJSONInputTracksAbsentAndNull(t *testing.T) {
	req := httptest.NewRequest(
		"PATCH",
		"/users/1",
		strings.NewReader(`{"name": null, "age": 25, "birth_date": "2026-01-02"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	input, err := DecodeJSONInput[*testInputModel](req)
	if err != nil {
		t.Fatalf("DecodeJSONInput() error = %v", err)
	}

	if input.Model == nil {
		t.Fatal("DecodeJSONInput() returned nil model")
	}

	if !input.AbsentFields.IsAbsent("id") {
		t.Fatal("id should be marked absent")
	}

	if input.AbsentFields.IsAbsent("name") {
		t.Fatal("name should be present even when it is null")
	}

	if input.Model.Name != nil {
		t.Fatalf("name should be nil for explicit JSON null, got %#v", *input.Model.Name)
	}

	if input.Model.Age == nil || *input.Model.Age != 25 {
		t.Fatalf("age = %#v, want 25", input.Model.Age)
	}

	if input.Model.BirthDate == nil {
		t.Fatal("birth_date should be parsed")
	}

	wantDate := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !input.Model.BirthDate.Equal(wantDate) {
		t.Fatalf("birth_date = %s, want %s", input.Model.BirthDate, wantDate)
	}
}

func TestDecodeURLValuesInput(t *testing.T) {
	values := url.Values{}
	values.Set("name", "Alice")
	values.Set("age", "30")
	values.Set("active", "on")
	values.Set("birth_date", "2026-01-02")

	input, err := DecodeURLValuesInput[*testInputModel](values)
	if err != nil {
		t.Fatalf("DecodeURLValuesInput() error = %v", err)
	}

	if input.Model.Name == nil || *input.Model.Name != "Alice" {
		t.Fatalf("name = %#v, want Alice", input.Model.Name)
	}

	if input.Model.Active == nil || !*input.Model.Active {
		t.Fatalf("active = %#v, want true", input.Model.Active)
	}

	if !input.AbsentFields.IsAbsent("id") {
		t.Fatal("id should be absent")
	}
}

func TestValidateModelInputRequiredAbsentOnInsert(t *testing.T) {
	name := "Alice"
	input := ModelInput[*testInputModel]{
		Model: &testInputModel{
			Name: &name,
		},
		AbsentFields: NewAbsentFieldSet(),
	}
	input.AbsentFields.SetAbsent("age")

	err := ValidateModelInput(input, true)
	if err == nil {
		t.Fatal("ValidateModelInput() error = nil, want required error")
	}

	got := TranslateError(LanguageEN, err)
	if !strings.Contains(got, "Age") {
		t.Fatalf("translated error = %q, want field alias", got)
	}
}

func TestBindUpdateModelInputBindsExplicitNullAndSkipsAbsent(t *testing.T) {
	id := 10
	input := ModelInput[*testInputModel]{
		Model:        &testInputModel{},
		AbsentFields: NewAbsentFieldSet(),
	}
	input.AbsentFields.SetAbsent("id")
	input.AbsentFields.SetAbsent("age")
	input.AbsentFields.SetAbsent("active")
	input.AbsentFields.SetAbsent("birth_date")

	keyModel := &testInputKeyModel{ID: &id}
	dbUpdate := pg.NewPgUpdate(input.Model)

	if err := BindUpdateModelInput(keyModel, input, dbUpdate); err != nil {
		t.Fatalf("BindUpdateModelInput() error = %v", err)
	}

	params := []any{}
	sql := dbUpdate.SQL(&params)
	if !strings.Contains(sql, "name = $1") {
		t.Fatalf("sql = %q, want name assignment", sql)
	}

	if len(params) == 0 || params[0] != nil {
		t.Fatalf("params[0] = %#v, want nil for explicit null", params)
	}
}

func TestBindInsertModelInputBindsExplicitNull(t *testing.T) {
	age := 25
	input := ModelInput[*testInputModel]{
		Model: &testInputModel{
			Age: &age,
		},
		AbsentFields: NewAbsentFieldSet(),
	}
	input.AbsentFields.SetAbsent("id")
	input.AbsentFields.SetAbsent("active")
	input.AbsentFields.SetAbsent("birth_date")

	dbInsert := pg.NewPgInsert(input.Model)
	if err := BindInsertModelInput(input, dbInsert); err != nil {
		t.Fatalf("BindInsertModelInput() error = %v", err)
	}

	params := []any{}
	sql := dbInsert.SQL(&params)
	if !strings.Contains(sql, "name") {
		t.Fatalf("sql = %q, want name field", sql)
	}

	if len(params) < 2 {
		t.Fatalf("params = %#v, want at least name and age", params)
	}

	if params[0] != nil {
		t.Fatalf("params[0] = %#v, want nil for explicit null name", params[0])
	}
}

func TestDecodeInvalidJSONValueReturnsValidationError(t *testing.T) {
	req := httptest.NewRequest("PATCH", "/users/1", strings.NewReader(`{"age":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")

	_, err := DecodeJSONInput[*testInputModel](req)
	if err == nil {
		t.Fatal("DecodeJSONInput() error = nil, want error")
	}

	if !strings.Contains(TranslateError(LanguageEN, err), "age") {
		t.Fatalf("translated error = %q, want age", TranslateError(LanguageEN, err))
	}

}
