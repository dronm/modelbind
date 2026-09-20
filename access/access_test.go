package access

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPredicateValidation(t *testing.T) {
	tests := []struct {
		name      string
		predicate Predicate
		valid     bool
	}{
		{"zero", Predicate{}, false},
		{"true", True(), true},
		{"false", False(), true},
		{"equal", Eq("customer_id", 42), true},
		{"not_equal", NotEq("status", "draft"), true},
		{"null", Eq("customer_id", (*int)(nil)), true},
		{"not_null", NotEq("customer_id", nil), true},
		{"less", Less("quantity", 10), true},
		{"less_equal", LessOrEqual("quantity", 10), true},
		{"greater", Greater("quantity", 10), true},
		{"greater_equal", GreaterOrEqual("quantity", 10), true},
		{"less_null", Less("quantity", nil), false},
		{"unsupported_operator", Compare("quantity", Operator("= 1 OR TRUE --"), 10), false},
		{"empty_field", Eq("", 1), false},
		{"sql_field", Eq("id OR TRUE", 1), false},
		{"alias_is_not_logical_id", Eq("o.id", 1), false},
		{"in", In("id", 1, 2), true},
		{"empty_in", In("id"), true},
		{"null_in", In("id", nil), false},
		{"slice_not_scalar", In("id", []int{1, 2}), false},
		{"nil_child", And(Eq("id", 1), Predicate{}), false},
		{"empty_and", And(), false},
		{"empty_or", Or(), false},
		{"group", And(Or(Eq("id", 1), Eq("id", 2)), Eq("customer_id", 42)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.predicate.Validate(); (err == nil) != tt.valid {
				t.Fatalf("Validate() = %v, valid = %v", err, tt.valid)
			}
		})
	}
	if Eq("id", nil).Kind() != KindIsNull || NotEq("id", nil).Kind() != KindIsNotNull {
		t.Fatal("NULL equality must use SQL null predicates")
	}
	deep := Eq("id", 1)
	for i := 0; i < 66; i++ {
		deep = And(deep)
	}
	if !errors.Is(deep.Validate(), ErrInvalidPredicate) {
		t.Fatal("expected excessive nesting to fail")
	}
}

func TestPredicateSnapshotsAndFields(t *testing.T) {
	id := 42
	values := []any{&id, 7}
	p := In("customer_id", values...)
	id = 99
	values[1] = 999
	got := p.Values()
	if !reflect.DeepEqual(got, []any{int64(42), int64(7)}) {
		t.Fatalf("values were not snapshotted: %#v", got)
	}
	got[0] = int64(100)
	if p.Values()[0] != int64(42) {
		t.Fatal("Values exposed internal slice")
	}
	children := []Predicate{p, Eq("id", 10)}
	group := And(children...)
	children[0] = True()
	copy := group.Children()
	copy[0] = False()
	if !reflect.DeepEqual(group.Fields(), []string{"customer_id", "id"}) {
		t.Fatalf("unexpected fields: %#v", group.Fields())
	}
}

func TestValuesEqual(t *testing.T) {
	type status string
	id := 42
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		a, b any
		want bool
	}{
		{int(42), int64(42), true},
		{&id, uint8(42), true},
		{uint64(math.MaxUint64), int64(-1), false},
		{uint64(math.MaxInt64), int64(math.MaxInt64), true},
		{int64(9007199254740993), int64(9007199254740992), false},
		{42, "42", false},
		{42, float64(42), false},
		{status("draft"), "draft", true},
		{nil, (*int)(nil), true},
		{nil, 0, false},
		{false, 0, false},
		{now, now.In(time.FixedZone("test", 7200)), true},
		{now, now.Add(time.Second), false},
		{now, "2026-01-02", false},
	}
	for _, tt := range tests {
		got, err := ValuesEqual(tt.a, tt.b)
		if err != nil || got != tt.want {
			t.Errorf("ValuesEqual(%#v, %#v) = %v, %v; want %v", tt.a, tt.b, got, err, tt.want)
		}
	}
	for _, value := range []any{math.NaN(), math.Inf(1), []int{1}, map[string]int{"a": 1}, struct{}{}, func() {}} {
		if _, err := NormalizeValue(value); !errors.Is(err, ErrUnsupportedValue) {
			t.Errorf("expected unsupported value for %T, got %v", value, err)
		}
	}
	var cyclic any
	cyclic = &cyclic
	if _, err := NormalizeValue(cyclic); !errors.Is(err, ErrUnsupportedValue) {
		t.Fatal("cyclic interface/pointer should fail")
	}
}

func TestWriteRulesPresenceSemantics(t *testing.T) {
	tests := []struct {
		name      string
		op        Operation
		rule      WriteRule
		submitted map[string]any
		want      map[string]any
		wantErr   error
	}{
		{"default_absent", Insert, Default(42), nil, map[string]any{"owner": int64(42)}, nil},
		{"default_present", Insert, Default(42), map[string]any{"owner": 7}, map[string]any{"owner": int64(7)}, nil},
		{"default_null", Insert, Default(42), map[string]any{"owner": nil}, map[string]any{"owner": nil}, nil},
		{"enforce_absent", Insert, Enforce(42), nil, map[string]any{"owner": int64(42)}, nil},
		{"enforce_equal", Insert, Enforce(42), map[string]any{"owner": int32(42)}, map[string]any{"owner": int64(42)}, nil},
		{"enforce_conflict", Insert, Enforce(42), map[string]any{"owner": 7}, nil, ErrWriteConflict},
		{"enforce_null", Insert, Enforce(42), map[string]any{"owner": nil}, nil, ErrWriteConflict},
		{"enforce_explicit_null", Insert, Enforce(nil), map[string]any{"owner": nil}, map[string]any{"owner": nil}, nil},
		{"update_enforce_absent", Update, Enforce(42), nil, map[string]any{}, nil},
		{"update_enforce_equal", Update, Enforce(42), map[string]any{"owner": 42}, map[string]any{"owner": int64(42)}, nil},
		{"update_enforce_conflict", Update, Enforce(42), map[string]any{"owner": 7}, nil, ErrWriteConflict},
		{"allowed_absent_insert", Insert, Allowed(3, 8), nil, nil, ErrWriteConflict},
		{"allowed_member", Insert, Allowed(3, 8), map[string]any{"owner": 8}, map[string]any{"owner": int64(8)}, nil},
		{"allowed_outside", Insert, Allowed(3, 8), map[string]any{"owner": 12}, nil, ErrWriteConflict},
		{"allowed_null_denied", Insert, Allowed(3, 8), map[string]any{"owner": nil}, nil, ErrWriteConflict},
		{"allowed_null", Insert, Allowed(3, nil), map[string]any{"owner": nil}, map[string]any{"owner": nil}, nil},
		{"allowed_empty", Insert, Allowed(), map[string]any{"owner": 3}, nil, ErrWriteConflict},
		{"allowed_update_absent", Update, Allowed(3, 8), nil, map[string]any{}, nil},
		{"allowed_update_conflict", Update, Allowed(3, 8), map[string]any{"owner": 9}, nil, ErrWriteConflict},
		{"immutable_absent", Update, Immutable(), nil, map[string]any{}, nil},
		{"immutable_present", Update, Immutable(), map[string]any{"owner": 42}, nil, ErrWriteConflict},
		{"immutable_null", Update, Immutable(), map[string]any{"owner": nil}, nil, ErrWriteConflict},
		{"default_update_invalid", Update, Default(42), nil, nil, ErrInvalidRule},
		{"immutable_insert_invalid", Insert, Immutable(), nil, nil, ErrInvalidRule},
		{"zero_rule", Insert, WriteRule{}, nil, nil, ErrInvalidRule},
		{"bad_rule_value", Insert, Enforce([]int{1}), nil, nil, ErrInvalidRule},
		{"unsupported_submitted", Insert, Enforce(42), map[string]any{"owner": []int{42}}, nil, ErrWriteConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (WriteRules{"owner": tt.rule}).Apply(tt.op, tt.submitted)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Apply() error = %v; want %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Apply() = %#v; want %#v", got, tt.want)
			}
		})
	}
}

func TestWriteRulesSnapshotsAndErrors(t *testing.T) {
	id := 987654
	rules := WriteRules{"owner": Enforce(&id), "status": Default("draft")}
	id = 12
	original := map[string]any{"title": "example"}
	effective, err := rules.Apply(Insert, original)
	if err != nil {
		t.Fatal(err)
	}
	if effective["owner"] != int64(987654) || len(original) != 1 {
		t.Fatal("policy/request state was mutated")
	}
	copy := rules.Clone()
	copy["owner"] = Enforce(99)
	_, err = rules.Apply(Insert, map[string]any{"owner": 12})
	var fieldError *FieldError
	if !errors.As(err, &fieldError) || fieldError.Field != "owner" || fieldError.Code != "enforced_value" {
		t.Fatalf("expected structured field error: %v", err)
	}
	if strings.Contains(err.Error(), "987654") {
		t.Fatal("error disclosed enforced value")
	}
	if err := (WriteRules{"owner_id;--": Enforce(1)}).Validate(Insert); !errors.Is(err, ErrInvalidField) {
		t.Fatal("invalid field accepted")
	}
	if err := rules.Validate(Read); !errors.Is(err, ErrInvalidRule) {
		t.Fatal("read operation accepted write rules")
	}
}

func TestPolicyValidation(t *testing.T) {
	update := Scope(Eq("owner", 42))
	update.Writes = WriteRules{"owner": Enforce(42)}
	tests := []struct {
		name   string
		policy Policy
		op     Operation
		valid  bool
	}{
		{"zero", Policy{}, Read, false},
		{"allow_read", AllowAll(), Read, true},
		{"allow_insert", AllowAll(), Insert, true},
		{"deny", DenyAll(), Delete, true},
		{"unknown_operation", AllowAll(), Operation("merge"), false},
		{"restricted_read", Scope(Eq("owner", 42)), Read, true},
		{"restricted_delete", Scope(In("owner", 42, 7)), Delete, true},
		{"restricted_update", update, Update, true},
		{"insert_rules", ForInsert(WriteRules{"owner": Enforce(42)}), Insert, true},
		{"empty_insert", ForInsert(nil), Insert, false},
		{"insert_rows", Scope(Eq("owner", 42)), Insert, false},
		{"read_missing_rows", ForInsert(WriteRules{"owner": Enforce(42)}), Read, false},
		{"read_writes", update, Read, false},
		{"delete_writes", update, Delete, false},
		{"ambiguous_unrestricted", Policy{Decision: Unrestricted, Rows: True()}, Read, false},
		{"ambiguous_denied", Policy{Decision: Denied, Writes: WriteRules{"owner": Enforce(42)}}, Insert, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate(tt.op)
			if (err == nil) != tt.valid {
				t.Fatalf("Validate() = %v, want valid %v", err, tt.valid)
			}
		})
	}
	copy := update.Clone()
	delete(copy.Writes, "owner")
	if len(update.Writes) != 1 {
		t.Fatal("policy clone shares rule map")
	}
}
