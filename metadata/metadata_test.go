package metadata

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func ptr[T any](v T) *T {
	return &v
}

func requireMessageError(t *testing.T, err error, id MessageID) *MessageError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error %q, got nil", id)
	}

	var msgErr *MessageError
	if !errors.As(err, &msgErr) {
		t.Fatalf("expected *MessageError, got %T: %v", err, err)
	}

	if msgErr.MessageID() != id {
		t.Fatalf("expected message ID %q, got %q", id, msgErr.MessageID())
	}

	return msgErr
}

func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireSetFlag(t *testing.T, got bool, want bool) {
	t.Helper()

	if got != want {
		t.Fatalf("expected set flag %v, got %v", want, got)
	}
}

func TestTranslate(t *testing.T) {
	data := map[string]any{
		"Field": "name",
	}

	t.Run("english", func(t *testing.T) {
		got := Translate(LanguageEN, MsgValueRequired, data)
		want := "field 'name', value is required"

		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("russian", func(t *testing.T) {
		got := Translate(LanguageRU, MsgValueRequired, data)
		want := "поле «name» обязательно для заполнения"

		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("language normalization", func(t *testing.T) {
		got := Translate(Language("ru-RU"), MsgValueRequired, data)
		want := "поле «name» обязательно для заполнения"

		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("unknown language falls back to english", func(t *testing.T) {
		got := Translate(Language("de-DE"), MsgValueRequired, data)
		want := "field 'name', value is required"

		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})
}

func TestTranslateError(t *testing.T) {
	err := NewMessageError(
		MsgValidationCast,
		map[string]any{
			"Field": "age",
			"Type":  "int64",
		},
	)

	got := TranslateError(LanguageRU, err)
	want := "значение поля «age» нельзя привести к типу int64"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	plainErr := errors.New("plain error")
	if got := TranslateError(LanguageRU, plainErr); got != plainErr.Error() {
		t.Fatalf("expected plain error text, got %q", got)
	}

	if got := TranslateError(LanguageRU, nil); got != "" {
		t.Fatalf("expected empty string for nil error, got %q", got)
	}
}

func TestFieldMetadataValidateRequired(t *testing.T) {
	validator := FieldMetadata{
		id:       "name",
		required: true,
	}

	t.Run("nil pointer required", func(t *testing.T) {
		var value *string

		err := validator.ValidateRequired(reflect.ValueOf(value))
		msgErr := requireMessageError(t, err, MsgValueRequired)

		if msgErr.Data["Field"] != "name" {
			t.Fatalf("expected field %q, got %#v", "name", msgErr.Data["Field"])
		}
	})

	t.Run("non nil pointer required", func(t *testing.T) {
		value := ptr("abc")
		requireNoError(t, validator.ValidateRequired(reflect.ValueOf(value)))
	})
}

func TestFieldBoolMetadataValidate(t *testing.T) {
	validator := NewFieldBoolMedata("Active", "active")

	t.Run("bool false is still set", func(t *testing.T) {
		set, err := validator.Validate(reflect.ValueOf(false))
		requireNoError(t, err)
		requireSetFlag(t, set, true)
	})

	t.Run("nil bool pointer is not set", func(t *testing.T) {
		var value *bool

		set, err := validator.Validate(reflect.ValueOf(value))
		requireNoError(t, err)
		requireSetFlag(t, set, false)
	})

	t.Run("invalid type", func(t *testing.T) {
		set, err := validator.Validate(reflect.ValueOf("true"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValidationCast)
	})
}

func TestFieldTextMetadataValidate(t *testing.T) {
	t.Run("valid text", func(t *testing.T) {
		validator := NewFieldTextMedata("Name", "name")
		validator.minLength = ptr[int64](2)
		validator.maxLength = ptr[int64](5)

		set, err := validator.Validate(reflect.ValueOf("John"))
		requireNoError(t, err)
		requireSetFlag(t, set, true)
	})

	t.Run("nil pointer is not set", func(t *testing.T) {
		validator := NewFieldTextMedata("Name", "name")
		var value *string

		set, err := validator.Validate(reflect.ValueOf(value))
		requireNoError(t, err)
		requireSetFlag(t, set, false)
	})

	t.Run("invalid type", func(t *testing.T) {
		validator := NewFieldTextMedata("Name", "name")

		set, err := validator.Validate(reflect.ValueOf(123))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValidationCast)
	})

	t.Run("too short", func(t *testing.T) {
		validator := NewFieldTextMedata("Name", "name")
		validator.minLength = ptr[int64](3)

		set, err := validator.Validate(reflect.ValueOf("ab"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgTextTooShort)
	})

	t.Run("too long", func(t *testing.T) {
		validator := NewFieldTextMedata("Name", "name")
		validator.maxLength = ptr[int64](3)

		set, err := validator.Validate(reflect.ValueOf("abcd"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgTextTooLong)
	})

	t.Run("fixed length", func(t *testing.T) {
		validator := NewFieldTextMedata("Code", "code")
		validator.fixLength = ptr[int64](4)

		set, err := validator.Validate(reflect.ValueOf("abc"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgTextFixedLength)
	})

	t.Run("regexp", func(t *testing.T) {
		validator := NewFieldTextMedata("Code", "code")
		validator.regExp = `^[A-Z]{2}$`

		set, err := validator.Validate(reflect.ValueOf("ab"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgTextRegExp)
	})

	t.Run("invalid regexp", func(t *testing.T) {
		validator := NewFieldTextMedata("Code", "code")
		validator.regExp = `[`

		set, err := validator.Validate(reflect.ValueOf("ab"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValidation)
	})

	t.Run("value list", func(t *testing.T) {
		validator := NewFieldTextMedata("Status", "status")
		validator.valList = []string{"new", "done"}

		set, err := validator.Validate(reflect.ValueOf("cancelled"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgTextValueList)
	})
}

func TestFieldIntMetadataValidate(t *testing.T) {
	t.Run("valid int", func(t *testing.T) {
		validator := NewFieldIntMedata("Age", "age")
		validator.minValue = ptr[int64](18)
		validator.maxValue = ptr[int64](99)

		set, err := validator.Validate(reflect.ValueOf(33))
		requireNoError(t, err)
		requireSetFlag(t, set, true)
	})

	t.Run("nil pointer is not set", func(t *testing.T) {
		validator := NewFieldIntMedata("Age", "age")
		var value *int64

		set, err := validator.Validate(reflect.ValueOf(value))
		requireNoError(t, err)
		requireSetFlag(t, set, false)
	})

	t.Run("too small", func(t *testing.T) {
		validator := NewFieldIntMedata("Age", "age")
		validator.minValue = ptr[int64](18)

		set, err := validator.Validate(reflect.ValueOf(int64(17)))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValueTooSmall)
	})

	t.Run("too big", func(t *testing.T) {
		validator := NewFieldIntMedata("Age", "age")
		validator.maxValue = ptr[int64](99)

		set, err := validator.Validate(reflect.ValueOf(int64(100)))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValueTooBig)
	})

	t.Run("invalid type", func(t *testing.T) {
		validator := NewFieldIntMedata("Age", "age")

		set, err := validator.Validate(reflect.ValueOf("10"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValidationCast)
	})

	t.Run("valid slice", func(t *testing.T) {
		validator := NewFieldIntMedata("IDs", "ids")
		validator.minValue = ptr[int64](1)

		set, err := validator.Validate(reflect.ValueOf([]int{1, 2, 3}))
		requireNoError(t, err)
		requireSetFlag(t, set, true)
	})

	t.Run("slice item validation", func(t *testing.T) {
		validator := NewFieldIntMedata("IDs", "ids")
		validator.minValue = ptr[int64](1)

		set, err := validator.Validate(reflect.ValueOf([]int{1, 0, 3}))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValueTooSmall)
	})

	t.Run("not slice", func(t *testing.T) {
		validator := NewFieldIntMedata("IDs", "ids")

		err := validator.ValidateSlice("ids", reflect.ValueOf(1))
		requireMessageError(t, err, MsgFieldNotSlice)
	})
}

func TestFieldFloatMetadataValidate(t *testing.T) {
	t.Run("valid float", func(t *testing.T) {
		validator := NewFieldFloatMedata("Price", "price")
		validator.minValue = ptr[float64](1.5)
		validator.maxValue = ptr[float64](10.5)
		validator.precision = 2

		set, err := validator.Validate(reflect.ValueOf(7.25))
		requireNoError(t, err)
		requireSetFlag(t, set, true)
	})

	t.Run("nil pointer is not set", func(t *testing.T) {
		validator := NewFieldFloatMedata("Price", "price")
		var value *float64

		set, err := validator.Validate(reflect.ValueOf(value))
		requireNoError(t, err)
		requireSetFlag(t, set, false)
	})

	t.Run("too small", func(t *testing.T) {
		validator := NewFieldFloatMedata("Price", "price")
		validator.minValue = ptr[float64](1.5)

		set, err := validator.Validate(reflect.ValueOf(1.49))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValueTooSmall)
	})

	t.Run("too big", func(t *testing.T) {
		validator := NewFieldFloatMedata("Price", "price")
		validator.maxValue = ptr[float64](10.5)

		set, err := validator.Validate(reflect.ValueOf(10.51))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValueTooBig)
	})

	t.Run("precision", func(t *testing.T) {
		validator := NewFieldFloatMedata("Price", "price")
		validator.precision = 2

		set, err := validator.Validate(reflect.ValueOf(1.234))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgFloatPrecision)
	})

	t.Run("invalid type", func(t *testing.T) {
		validator := NewFieldFloatMedata("Price", "price")

		set, err := validator.Validate(reflect.ValueOf("1.5"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValidationCast)
	})
}

func TestFieldDateMetadataValidate(t *testing.T) {
	t.Run("zero time is not set", func(t *testing.T) {
		validator := NewFieldDateMedata("CreatedAt", "created_at", FieldTypeDatetime)

		set, err := validator.Validate(reflect.ValueOf(time.Time{}))
		requireNoError(t, err)
		requireSetFlag(t, set, false)
	})

	t.Run("non zero time is set", func(t *testing.T) {
		validator := NewFieldDateMedata("CreatedAt", "created_at", FieldTypeDatetime)

		set, err := validator.Validate(reflect.ValueOf(time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)))
		requireNoError(t, err)
		requireSetFlag(t, set, true)
	})

	t.Run("nil pointer is not set", func(t *testing.T) {
		validator := NewFieldDateMedata("CreatedAt", "created_at", FieldTypeDatetime)
		var value *time.Time

		set, err := validator.Validate(reflect.ValueOf(value))
		requireNoError(t, err)
		requireSetFlag(t, set, false)
	})

	t.Run("valid date string", func(t *testing.T) {
		validator := NewFieldDateMedata("Date", "date", FieldTypeDate)

		set, err := validator.Validate(reflect.ValueOf("2026-05-03"))
		requireNoError(t, err)
		requireSetFlag(t, set, true)
	})

	t.Run("invalid date string length", func(t *testing.T) {
		validator := NewFieldDateMedata("Date", "date", FieldTypeDate)

		set, err := validator.Validate(reflect.ValueOf("2026-05"))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValidationCast)
	})

	t.Run("invalid type", func(t *testing.T) {
		validator := NewFieldDateMedata("Date", "date", FieldTypeDate)

		set, err := validator.Validate(reflect.ValueOf(123))
		requireSetFlag(t, set, true)
		requireMessageError(t, err, MsgValidationCast)
	})
}

func TestFieldAnnotationValue(t *testing.T) {
	type model struct {
		Name       string `json:"name,omitempty,string"`
		Ignored    string `json:"-,omitempty"`
		EmptyName  string `json:",omitempty"`
		CustomName string `db:"custom,column"`
	}

	modelType := reflect.TypeOf(model{})

	tests := []struct {
		fieldName string
		tagName   string
		want      string
	}{
		{fieldName: "Name", tagName: "json", want: "name"},
		{fieldName: "Ignored", tagName: "json", want: "-"},
		{fieldName: "EmptyName", tagName: "json", want: ""},
		{fieldName: "CustomName", tagName: "db", want: "custom,column"},
	}

	for _, test := range tests {
		t.Run(test.fieldName+"/"+test.tagName, func(t *testing.T) {
			field, ok := modelType.FieldByName(test.fieldName)
			if !ok {
				t.Fatalf("field %q not found", test.fieldName)
			}

			if got := FieldAnnotationValue(field, test.tagName); got != test.want {
				t.Fatalf("FieldAnnotationValue() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNewModelMetadata(t *testing.T) {
	type product struct {
		Name string `json:"name,omitempty" alias:"Product name" required:"" max:"10" min:"2" role:"public"`
		Age  int64  `json:"age" min:"18" max:"99"`
		Skip string `json:"-"`
	}

	oldControlledTags := ControledTags
	ControledTags = []string{"role"}
	t.Cleanup(func() {
		ControledTags = oldControlledTags
	})

	meta, err := NewModelMetadata(product{})
	requireNoError(t, err)

	if meta.ID == "" {
		t.Fatal("expected metadata ID")
	}

	if _, ok := meta.Fields["name"]; !ok {
		t.Fatal("expected name field metadata")
	}

	if _, ok := meta.Fields["name,omitempty"]; ok {
		t.Fatal("did not expect JSON options in metadata field name")
	}

	if len(meta.FieldTagList) == 0 || meta.FieldTagList[0] != "name" {
		t.Fatalf("FieldTagList = %#v, want first field name %q", meta.FieldTagList, "name")
	}

	nameField := meta.Fields["name"]
	if nameField.Descr() != "Product name" {
		t.Fatalf("expected alias %q, got %q", "Product name", nameField.Descr())
	}

	if !nameField.Required() {
		t.Fatal("expected name field to be required")
	}

	if _, ok := meta.Fields["Skip"]; ok {
		t.Fatal("did not expect skipped field metadata")
	}

	if got := meta.Tags["Name"]["role"]; got != "public" {
		t.Fatalf("expected controlled tag value %q, got %q", "public", got)
	}
}

func TestValidateModel(t *testing.T) {
	t.Run("valid model", func(t *testing.T) {
		type request struct {
			Name string `json:"name" required:"" min:"2"`
		}

		requireNoError(t, ValidateModel(request{Name: "John"}, "json"))
	})

	t.Run("required pointer", func(t *testing.T) {
		type request struct {
			Name *string `json:"name" required:""`
		}

		err := ValidateModel(request{}, "json")
		requireMessageError(t, err, MsgValueRequired)
	})

	t.Run("invalid model", func(t *testing.T) {
		err := ValidateModel(123, "json")
		requireMessageError(t, err, MsgModelNotPointerOrStruct)
	})
}

func TestMessageTemplateCoverage(t *testing.T) {
	for id := range DefaultMessageTemplates {
		if _, ok := RussianMessageTemplates[id]; !ok {
			t.Fatalf("missing russian message template for %q", id)
		}
	}

	for id := range RussianMessageTemplates {
		if _, ok := DefaultMessageTemplates[id]; !ok {
			t.Fatalf("missing english message template for %q", id)
		}
	}
}

func TestUnknownMessageID(t *testing.T) {
	got := Translate(LanguageEN, MessageID("unknown.message"), nil)
	want := "unknown.message"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestErrorStringIsEnglish(t *testing.T) {
	err := NewMessageError(
		MsgValidationCast,
		map[string]any{
			"Field": "amount",
			"Type":  "float64",
		},
	)

	if !strings.Contains(err.Error(), "amount") || !strings.Contains(err.Error(), "float64") {
		t.Fatalf("expected english error string to contain data, got %q", err.Error())
	}
}
