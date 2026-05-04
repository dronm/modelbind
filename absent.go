package modelbind

import "sort"

// AbsentFieldSet stores model field IDs that were not present in the input.
//
// Field IDs are the public field names used by modelbind metadata, usually the
// json tag value, not the Go struct field name.
type AbsentFieldSet struct {
	fields map[string]struct{}
}

func NewAbsentFieldSet() AbsentFieldSet {
	return AbsentFieldSet{
		fields: make(map[string]struct{}),
	}
}

func (s *AbsentFieldSet) SetAbsent(field string) {
	if s.fields == nil {
		s.fields = make(map[string]struct{})
	}

	s.fields[field] = struct{}{}
}

func (s AbsentFieldSet) IsAbsent(field string) bool {
	if s.fields == nil {
		return false
	}

	_, ok := s.fields[field]
	return ok
}

func (s AbsentFieldSet) IsPresent(field string) bool {
	return !s.IsAbsent(field)
}

func (s AbsentFieldSet) IsTracked() bool {
	return s.fields != nil
}

func (s AbsentFieldSet) Len() int {
	return len(s.fields)
}

func (s AbsentFieldSet) Fields() []string {
	res := make([]string, 0, len(s.fields))
	for field := range s.fields {
		res = append(res, field)
	}

	sort.Strings(res)
	return res
}

// ModelInput combines a decoded model with field-presence metadata.
//
// For pointer fields, a nil pointer means either explicit null or absent. Use
// AbsentFields to distinguish these states:
//   - AbsentFields.IsAbsent("name") == true: field was not sent.
//   - pointer == nil and field is not absent: field was explicitly sent as null.
type ModelInput[T any] struct {
	Model        T
	AbsentFields AbsentFieldSet
}

func (in ModelInput[T]) IsAbsent(field string) bool {
	return in.AbsentFields.IsAbsent(field)
}

func (in ModelInput[T]) IsPresent(field string) bool {
	return !in.AbsentFields.IsTracked() || !in.AbsentFields.IsAbsent(field)
}

func (in ModelInput[T]) Validate(forInsert bool) error {
	return ValidateModelInput(in, forInsert)
}
