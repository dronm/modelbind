package modelbind

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/dronm/modelbind/access"
	"github.com/dronm/modelbind/metadata"
	"github.com/dronm/modelbind/types"
)

var ErrMissingKey = errors.New("missing model key")

// PrepareModelInput applies scalar rules before required-field validation. Rule
// IDs must match input metadata IDs (usually JSON tags). The returned model is a
// shallow struct copy with an independent presence set and newly assigned scalar
// pointers; the submitted model and its original presence metadata are unchanged.
// Unmodified reference-valued fields remain shared and must not be mutated.
//
// Unknown fields, srvCalc fields, unsupported values and lossy conversions are
// rejected. Fields absent from the input Go type can instead be governed at the
// SQL-builder layer with an explicit PolicyBinding.Writes mapping.
func PrepareModelInput[T any](input ModelInput[T], operation access.Operation, rules access.WriteRules) (ModelInput[T], error) {
	if err := rules.Validate(operation); err != nil {
		return ModelInput[T]{}, err
	}
	modelType, modelValue, err := modelPointerValue(input.Model, "PrepareModelInput")
	if err != nil {
		return ModelInput[T]{}, err
	}
	md, err := metadata.NewModelMetadata(input.Model)
	if err != nil {
		return ModelInput[T]{}, err
	}
	indices := make(map[string]int)
	for i := 0; i < modelType.NumField(); i++ {
		field := metadata.FieldAnnotationValue(modelType.Field(i), metadata.FieldAnnotationName)
		if field == "" || field == "-" {
			continue
		}
		if _, duplicate := indices[field]; duplicate {
			return ModelInput[T]{}, &access.FieldError{Field: field, Code: "duplicate_input_field", Kind: access.ErrInvalidField}
		}
		indices[field] = i
	}
	submitted := make(map[string]any, len(rules))
	for _, field := range rules.Fields() {
		index, ok := indices[field]
		if !ok {
			return ModelInput[T]{}, &access.FieldError{Field: field, Code: "unknown_input_field", Kind: access.ErrInvalidField}
		}
		fieldMD := md.Fields[field]
		if fieldMD == nil || fieldMD.SrvCalc() {
			return ModelInput[T]{}, &access.FieldError{Field: field, Code: "database_generated_field", Kind: access.ErrInvalidField}
		}
		value := modelValue.Field(index)
		if !value.CanInterface() || !value.CanSet() {
			return ModelInput[T]{}, &access.FieldError{Field: field, Code: "inaccessible_input_field", Kind: access.ErrInvalidField}
		}
		if input.IsPresent(field) {
			submitted[field] = value.Interface()
		}
	}
	effective, err := rules.Apply(operation, submitted)
	if err != nil {
		return ModelInput[T]{}, err
	}
	copyPointer := reflect.New(modelType)
	copyPointer.Elem().Set(modelValue)
	presence := input.AbsentFields.Clone()
	for _, field := range rules.Fields() {
		value, present := effective[field]
		if !present {
			continue
		}
		target := copyPointer.Elem().Field(indices[field])
		converted, err := policyFieldValue(value, target.Type())
		if err != nil {
			return ModelInput[T]{}, fmt.Errorf("%w: field %q: %w", access.ErrInvalidRule, field, err)
		}
		target.Set(converted)
		presence.SetPresent(field)
	}
	// Normal pointer model types retain their dynamic type. Defined pointer
	// aliases can be converted without changing the underlying struct.
	originalType := reflect.TypeOf(input.Model)
	if copyPointer.Type() != originalType && copyPointer.CanConvert(originalType) {
		copyPointer = copyPointer.Convert(originalType)
	}
	model, ok := copyPointer.Interface().(T)
	if !ok {
		return ModelInput[T]{}, fmt.Errorf("%w: unsupported model pointer type", access.ErrInvalidRule)
	}
	return ModelInput[T]{Model: model, AbsentFields: presence}, nil
}

// policyFieldValue permits exact scalar conversion only. In particular integer
// overflow, float-to-int conversion and int-to-string conversion are rejected.
func policyFieldValue(value any, target reflect.Type) (reflect.Value, error) {
	result := reflect.New(target).Elem()
	if value == nil {
		switch target.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
			return result, nil
		default:
			return reflect.Value{}, fmt.Errorf("NULL cannot be assigned to %s", target)
		}
	}
	if target.Kind() == reflect.Pointer {
		inner, err := policyFieldValue(value, target.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		result.Set(reflect.New(target.Elem()))
		result.Elem().Set(inner)
		return result, nil
	}
	if target == reflect.TypeOf(time.Time{}) {
		if date, ok := value.(time.Time); ok {
			result.Set(reflect.ValueOf(date))
			return result, nil
		}
	}
	switch target.Kind() {
	case reflect.String:
		if v, ok := value.(string); ok {
			result.SetString(v)
			return result, nil
		}
	case reflect.Bool:
		if v, ok := value.(bool); ok {
			result.SetBool(v)
			return result, nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var n int64
		switch v := value.(type) {
		case int64:
			n = v
		case uint64:
			if v > math.MaxInt64 {
				return reflect.Value{}, fmt.Errorf("integer overflows %s", target)
			}
			n = int64(v)
		default:
			return reflect.Value{}, fmt.Errorf("expected integer for %s", target)
		}
		if !result.OverflowInt(n) {
			result.SetInt(n)
			return result, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var n uint64
		switch v := value.(type) {
		case uint64:
			n = v
		case int64:
			if v < 0 {
				return reflect.Value{}, fmt.Errorf("negative integer for %s", target)
			}
			n = uint64(v)
		default:
			return reflect.Value{}, fmt.Errorf("expected unsigned integer for %s", target)
		}
		if !result.OverflowUint(n) {
			result.SetUint(n)
			return result, nil
		}
	case reflect.Float32, reflect.Float64:
		if v, ok := value.(float64); ok && !result.OverflowFloat(v) {
			if target.Kind() == reflect.Float64 || float64(float32(v)) == v {
				result.SetFloat(v)
				return result, nil
			}
		}
	}
	return reflect.Value{}, fmt.Errorf("unsupported or lossy conversion to %s", target)
}

// RequireModelKeys checks that a key model has at least one annotated field and
// that ALL such fields are non-null. Zero numeric keys are not assumed invalid:
// application-specific key validation remains the caller's responsibility.
// This check is independent of any server row predicate.
func RequireModelKeys(keyModel any) error {
	modelType, modelValue, err := modelStructOrPointerValue(keyModel, "RequireModelKeys")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMissingKey, err)
	}
	count := 0
	for i := 0; i < modelType.NumField(); i++ {
		field := metadata.FieldAnnotationValue(modelType.Field(i), metadata.FieldAnnotationName)
		if field == "" || field == "-" {
			continue
		}
		count++
		value := modelValue.Field(i)
		if !value.CanInterface() || isNilReflectValue(value) {
			return fmt.Errorf("%w: field %q", ErrMissingKey, field)
		}
		if nullable, ok := value.Interface().(metadata.NullableField); ok {
			if !nullable.IsSet() || nullable.IsNull() {
				return fmt.Errorf("%w: field %q", ErrMissingKey, field)
			}
		}
		normalized, err := access.NormalizeValue(value.Interface())
		if err != nil {
			return fmt.Errorf("%w: field %q: %w", ErrMissingKey, field, err)
		}
		if normalized == nil {
			return fmt.Errorf("%w: field %q", ErrMissingKey, field)
		}
	}
	if count == 0 {
		return ErrMissingKey
	}
	return nil
}

// BindInsertModelInputWithPolicy requires a policy-aware builder with an already
// installed policy, prepares input, validates it, then binds it. Use the returned
// effective model for RETURNING/results; the submitted model stays unchanged.
// Policy field IDs here must match input metadata IDs. On error discard the
// builder, as with the existing binding APIs.
func BindInsertModelInputWithPolicy[T any](input ModelInput[T], builder types.DBInserter) (ModelInput[T], error) {
	rules, err := builderWriteRules(builder, access.Insert)
	if err != nil {
		return ModelInput[T]{}, err
	}
	effective, err := PrepareModelInput(input, access.Insert, rules)
	if err != nil {
		return ModelInput[T]{}, err
	}
	if err := effective.Validate(true); err != nil {
		return ModelInput[T]{}, err
	}
	if err := effective.BindInsert(builder); err != nil {
		return ModelInput[T]{}, err
	}
	return effective, nil
}

func BindUpdateModelInputWithPolicy[T any](keyModel any, input ModelInput[T], builder types.DBUpdater) (ModelInput[T], error) {
	if err := RequireModelKeys(keyModel); err != nil {
		return ModelInput[T]{}, err
	}
	rules, err := builderWriteRules(builder, access.Update)
	if err != nil {
		return ModelInput[T]{}, err
	}
	effective, err := PrepareModelInput(input, access.Update, rules)
	if err != nil {
		return ModelInput[T]{}, err
	}
	if err := effective.Validate(false); err != nil {
		return ModelInput[T]{}, err
	}
	if err := effective.BindUpdate(keyModel, builder); err != nil {
		return ModelInput[T]{}, err
	}
	return effective, nil
}

func builderWriteRules(builder any, operation access.Operation) (access.WriteRules, error) {
	aware, ok := builder.(types.AccessWritePolicy)
	if !ok || isNilReflectValue(reflect.ValueOf(builder)) {
		return nil, fmt.Errorf("%w: builder does not provide an installed write policy", access.ErrInvalidPolicy)
	}
	actual, rules, err := aware.AccessWriteRules()
	if err != nil {
		return nil, err
	}
	if actual != operation {
		return nil, fmt.Errorf("%w: builder operation mismatch", access.ErrInvalidPolicy)
	}
	return rules, nil
}
