package modelbind

import (
	"reflect"
	"testing"

	"github.com/dronm/modelbind/pg"
)

type testDefaultName string

const testDefaultNameRus testDefaultName = "rus"

type testDBValueModel struct {
	ID          int             `json:"id" primaryKey:"" srvCalc:""`
	DefaultName testDefaultName `json:"default_name" required:""`
	SeriesCodes []string        `json:"series_codes" required:""`
	Payload     map[string]any  `json:"payload"`
}

func (m *testDBValueModel) Relation() string {
	return "vehicle_test"
}

func TestBindInsertModelInputUsesPlainStringForNamedStringAndSliceForArray(t *testing.T) {
	input := ModelInput[*testDBValueModel]{
		Model: &testDBValueModel{
			DefaultName: testDefaultNameRus,
			SeriesCodes: []string{
				"XV70",
				"XV75",
			},
		},
		AbsentFields: NewAbsentFieldSet(),
	}
	input.AbsentFields.SetAbsent("id")
	input.AbsentFields.SetAbsent("payload")

	dbInsert := pg.NewPgInsert(input.Model)
	if err := BindInsertModelInput(input, dbInsert); err != nil {
		t.Fatalf("BindInsertModelInput() error = %v", err)
	}

	params := []any{}
	_ = dbInsert.SQL(&params)

	if len(params) != 2 {
		t.Fatalf("params = %#v, want default_name and series_codes", params)
	}

	if got, ok := params[0].(string); !ok || got != "rus" {
		t.Fatalf("params[0] = %#v (%T), want string rus", params[0], params[0])
	}

	wantSeries := []string{"XV70", "XV75"}
	if !reflect.DeepEqual(params[1], wantSeries) {
		t.Fatalf("params[1] = %#v (%T), want %#v", params[1], params[1], wantSeries)
	}
}
