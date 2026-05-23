package modelbind

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dronm/modelbind/metadata"
)

var ParseFormMaxMemory int64 = 32 << 20

// DecodeJSONInput decodes an incoming HTTP request into a model and tracks
// absent fields.
// JSON requests are decoded from body, form requests
// from form values, and requests without a form/json body from URL query values.
func DecodeJSONInput[T any](r *http.Request) (ModelInput[T], error) {
	return DecodeRequestInput[T](r)
}

func DecodeRequestInput[T any](r *http.Request) (ModelInput[T], error) {
	const funcName = "DecodeRequestInput"

	if r == nil {
		return ModelInput[T]{}, NewMessageError(
			MsgDecodeInvalidModel,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	mediaType := ""
	if contentType := strings.TrimSpace(r.Header.Get("Content-Type")); contentType != "" {
		parsedMediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil {
			mediaType = parsedMediaType
		}
	}

	switch mediaType {
	case "application/json", "text/json":
		return DecodeJSONBodyInput[T](r.Body)

	case "application/x-www-form-urlencoded", "multipart/form-data":
		if err := r.ParseMultipartForm(ParseFormMaxMemory); err != nil {
			if err := r.ParseForm(); err != nil {
				return ModelInput[T]{}, NewMessageError(
					MsgDecodeInvalidFieldValue,
					map[string]any{
						"Func":  funcName,
						"Field": "form",
						"Value": err.Error(),
					},
				)
			}
		}

		return DecodeURLValuesInput[T](r.Form)
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return ModelInput[T]{}, NewMessageError(
				MsgDecodeReadFailed,
				map[string]any{
					"Func":  funcName,
					"Error": err.Error(),
				},
			)
		}

		trimmed := bytes.TrimSpace(body)
		if len(trimmed) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(body))
			if trimmed[0] == '{' {
				return DecodeJSONBodyInput[T](r.Body)
			}
		}
	}

	return DecodeURLValuesInput[T](r.URL.Query())
}

func DecodeJSONBodyInput[T any](reader io.Reader) (ModelInput[T], error) {
	const funcName = "DecodeJSONBodyInput"

	body, err := io.ReadAll(reader)
	if err != nil {
		return ModelInput[T]{}, NewMessageError(
			MsgDecodeReadFailed,
			map[string]any{
				"Func":  funcName,
				"Error": err.Error(),
			},
		)
	}

	model, modelVal, err := newDecodeModel[T](funcName)
	if err != nil {
		return ModelInput[T]{}, err
	}

	modelMd, err := metadata.NewModelMetadata(model)
	if err != nil {
		return ModelInput[T]{}, err
	}

	rawFields := make(map[string]json.RawMessage)
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &rawFields); err != nil {
			return ModelInput[T]{}, NewMessageError(
				MsgDecodeJSONFailed,
				map[string]any{
					"Func":  funcName,
					"Error": err.Error(),
				},
			)
		}
	}

	absent := NewAbsentFieldSet()
	validationErr := &ValidationError{}

	for fieldID, fieldMd := range modelMd.Fields {
		rawValue, ok := rawFields[fieldID]
		if !ok {
			absent.SetAbsent(fieldID)
			continue
		}

		field := modelVal.FieldByName(fieldMd.ModelID())
		if !field.IsValid() || !field.CanSet() {
			validationErr.Add(NewMessageError(
				MsgDecodeUnsupportedField,
				map[string]any{
					"Func":  funcName,
					"Field": fieldID,
					"Type":  reflectValueTypeName(field),
				},
			))
			continue
		}

		if err := setFieldFromJSON(field, fieldMd, rawValue, funcName); err != nil {
			validationErr.Add(err)
		}
	}

	if validationErr.HasErrors() {
		return ModelInput[T]{}, validationErr
	}

	return ModelInput[T]{
		Model:        model,
		AbsentFields: absent,
	}, nil
}

func DecodeURLValuesInput[T any](values url.Values) (ModelInput[T], error) {
	const funcName = "DecodeURLValuesInput"

	model, modelVal, err := newDecodeModel[T](funcName)
	if err != nil {
		return ModelInput[T]{}, err
	}

	modelMd, err := metadata.NewModelMetadata(model)
	if err != nil {
		return ModelInput[T]{}, err
	}

	absent := NewAbsentFieldSet()
	validationErr := &ValidationError{}

	for fieldID, fieldMd := range modelMd.Fields {
		fieldValues, ok := values[fieldID]
		if !ok {
			absent.SetAbsent(fieldID)
			continue
		}

		field := modelVal.FieldByName(fieldMd.ModelID())
		if !field.IsValid() || !field.CanSet() {
			validationErr.Add(NewMessageError(
				MsgDecodeUnsupportedField,
				map[string]any{
					"Func":  funcName,
					"Field": fieldID,
					"Type":  reflectValueTypeName(field),
				},
			))
			continue
		}

		if err := setFieldFromStrings(field, fieldMd, fieldValues, funcName); err != nil {
			validationErr.Add(err)
		}
	}

	if validationErr.HasErrors() {
		return ModelInput[T]{}, validationErr
	}

	return ModelInput[T]{
		Model:        model,
		AbsentFields: absent,
	}, nil
}

func newDecodeModel[T any](funcName string) (T, reflect.Value, error) {
	var model T

	modelSlot := reflect.ValueOf(&model).Elem()
	modelType := modelSlot.Type()

	if modelType.Kind() == reflect.Pointer {
		if modelType.Elem().Kind() != reflect.Struct {
			return model, reflect.Value{}, NewMessageError(
				MsgDecodeInvalidModel,
				map[string]any{
					"Func": funcName,
				},
			)
		}

		newModel := reflect.New(modelType.Elem())
		modelSlot.Set(newModel)
		return model, newModel.Elem(), nil
	}

	if modelType.Kind() != reflect.Struct {
		return model, reflect.Value{}, NewMessageError(
			MsgDecodeInvalidModel,
			map[string]any{
				"Func": funcName,
			},
		)
	}

	return model, modelSlot, nil
}

func setFieldFromJSON(field reflect.Value, fieldMd metadata.FieldValidator, rawValue json.RawMessage, funcName string) error {
	if isJSONNull(rawValue) {
		return setFieldNull(field, fieldMd.ID(), funcName)
	}

	if isTimeValue(field) || isTimePointerValue(field) {
		var value string
		if err := json.Unmarshal(rawValue, &value); err != nil {
			return NewMessageError(
				MsgDecodeInvalidFieldValue,
				map[string]any{
					"Func":  funcName,
					"Field": fieldMd.ID(),
					"Value": string(rawValue),
				},
			)
		}

		return setTimeField(field, fieldMd, value, funcName)
	}

	if field.Kind() == reflect.Pointer {
		if field.Type().Elem().Kind() == reflect.Struct && field.Type().Elem() != reflect.TypeFor[time.Time]() {
			newValue := reflect.New(field.Type().Elem())
			if err := json.Unmarshal(rawValue, newValue.Interface()); err != nil {
				return decodeValueError(funcName, fieldMd.ID(), string(rawValue))
			}

			field.Set(newValue)
			return nil
		}

		newValue := reflect.New(field.Type().Elem())
		if err := json.Unmarshal(rawValue, newValue.Interface()); err != nil {
			return decodeValueError(funcName, fieldMd.ID(), string(rawValue))
		}

		field.Set(newValue)
		return nil
	}

	if err := json.Unmarshal(rawValue, field.Addr().Interface()); err != nil {
		return decodeValueError(funcName, fieldMd.ID(), string(rawValue))
	}

	return nil
}

func setFieldFromStrings(field reflect.Value, fieldMd metadata.FieldValidator, values []string, funcName string) error {
	if field.Kind() == reflect.Slice {
		return setSliceFieldFromStrings(field, fieldMd, values, funcName)
	}

	value := ""
	if len(values) > 0 {
		value = values[len(values)-1]
	}

	if isFormNull(value) {
		return setFieldNull(field, fieldMd.ID(), funcName)
	}

	return setScalarFieldFromString(field, fieldMd, value, funcName)
}

func setScalarFieldFromString(field reflect.Value, fieldMd metadata.FieldValidator, value string, funcName string) error {
	if field.Kind() == reflect.Pointer {
		newValue := reflect.New(field.Type().Elem())
		if err := setScalarFieldFromString(newValue.Elem(), fieldMd, value, funcName); err != nil {
			return err
		}

		field.Set(newValue)
		return nil
	}

	if isTimeValue(field) {
		return setTimeField(field, fieldMd, value, funcName)
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
		return nil

	case reflect.Bool:
		parsedValue, err := parseBool(value)
		if err != nil {
			return decodeValueError(funcName, fieldMd.ID(), value)
		}

		field.SetBool(parsedValue)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsedValue, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return decodeValueError(funcName, fieldMd.ID(), value)
		}

		field.SetInt(parsedValue)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsedValue, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return decodeValueError(funcName, fieldMd.ID(), value)
		}

		field.SetUint(parsedValue)
		return nil

	case reflect.Float32, reflect.Float64:
		parsedValue, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return decodeValueError(funcName, fieldMd.ID(), value)
		}

		field.SetFloat(parsedValue)
		return nil

	case reflect.Struct, reflect.Map, reflect.Interface:
		if err := json.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
			return decodeValueError(funcName, fieldMd.ID(), value)
		}

		return nil
	}

	return NewMessageError(
		MsgDecodeUnsupportedField,
		map[string]any{
			"Func":  funcName,
			"Field": fieldMd.ID(),
			"Type":  reflectValueTypeName(field),
		},
	)
}

func setSliceFieldFromStrings(field reflect.Value, fieldMd metadata.FieldValidator, values []string, funcName string) error {
	elemType := field.Type().Elem()
	flatValues := flattenFormValues(values)
	newSlice := reflect.MakeSlice(field.Type(), 0, len(flatValues))

	for _, value := range flatValues {
		elem := reflect.New(elemType).Elem()
		if elemType.Kind() == reflect.Pointer {
			elem = reflect.New(elemType.Elem())
			if isFormNull(value) {
				newSlice = reflect.Append(newSlice, reflect.Zero(elemType))
				continue
			}

			if err := setScalarFieldFromString(elem.Elem(), fieldMd, value, funcName); err != nil {
				return err
			}

			newSlice = reflect.Append(newSlice, elem)
			continue
		}

		if err := setScalarFieldFromString(elem, fieldMd, value, funcName); err != nil {
			return err
		}

		newSlice = reflect.Append(newSlice, elem)
	}

	field.Set(newSlice)
	return nil
}

func setFieldNull(field reflect.Value, fieldID string, funcName string) error {
	if field.Kind() == reflect.Pointer || field.Kind() == reflect.Interface || field.Kind() == reflect.Map || field.Kind() == reflect.Slice {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	return NewMessageError(
		MsgDecodeInvalidFieldValue,
		map[string]any{
			"Func":  funcName,
			"Field": fieldID,
			"Value": "null",
		},
	)
}

func setTimeField(field reflect.Value, fieldMd metadata.FieldValidator, value string, funcName string) error {
	if value == "" {
		return decodeValueError(funcName, fieldMd.ID(), value)
	}

	parsedValue, err := parseTimeByFieldType(value, fieldMd.DataType())
	if err != nil {
		return decodeValueError(funcName, fieldMd.ID(), value)
	}

	if field.Kind() == reflect.Pointer {
		newValue := reflect.New(field.Type().Elem())
		newValue.Elem().Set(reflect.ValueOf(parsedValue))
		field.Set(newValue)
		return nil
	}

	field.Set(reflect.ValueOf(parsedValue))
	return nil
}

func parseTimeByFieldType(value string, fieldType metadata.FieldDataType) (time.Time, error) {
	layouts := []string{time.RFC3339, time.RFC3339Nano}

	switch fieldType {
	case metadata.FieldTypeDate:
		layouts = []string{"2006-01-02"}

	case metadata.FieldTypeTime:
		layouts = []string{"15:04", "15:04:05"}

	case metadata.FieldTypeDatetime:
		layouts = []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			time.RFC3339,
			time.RFC3339Nano,
		}

	case metadata.FieldTypeDatetimeTZ:
		layouts = []string{time.RFC3339, time.RFC3339Nano}
	}

	var lastErr error
	for _, layout := range layouts {
		parsedValue, err := time.Parse(layout, value)
		if err == nil {
			return parsedValue, nil
		}

		lastErr = err
	}

	return time.Time{}, lastErr
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "yes", "y":
		return true, nil
	case "off", "no", "n":
		return false, nil
	}

	return strconv.ParseBool(value)
}

func flattenFormValues(values []string) []string {
	res := make([]string, 0, len(values))
	for _, value := range values {
		if strings.Contains(value, ",") {
			for part := range strings.SplitSeq(value, ",") {
				res = append(res, strings.TrimSpace(part))
			}
			continue
		}

		res = append(res, value)
	}

	return res
}

func isJSONNull(rawValue json.RawMessage) bool {
	return strings.EqualFold(strings.TrimSpace(string(rawValue)), "null")
}

func isFormNull(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "null")
}

func isTimeValue(field reflect.Value) bool {
	return field.IsValid() && field.Type() == reflect.TypeFor[time.Time]()
}

func isTimePointerValue(field reflect.Value) bool {
	return field.IsValid() && field.Kind() == reflect.Pointer && field.Type().Elem() == reflect.TypeFor[time.Time]()
}

func decodeValueError(funcName string, fieldID string, value string) error {
	return NewMessageError(
		MsgDecodeInvalidFieldValue,
		map[string]any{
			"Func":  funcName,
			"Field": fieldID,
			"Value": fmt.Sprintf("%q", value),
		},
	)
}

func reflectValueTypeName(field reflect.Value) string {
	if !field.IsValid() {
		return "<invalid>"
	}

	return field.Type().String()
}
