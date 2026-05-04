package pg

import "testing"

func TestSanitizeSQLFieldRef(t *testing.T) {
	tests := []struct {
		name  string
		field string
		valid bool
	}{
		{name: "plain", field: "field_name", valid: true},
		{name: "qualified", field: "t.field_name", valid: true},
		{name: "json key", field: "payload->>'name'", valid: true},
		{name: "json index", field: "payload->0", valid: true},
		{name: "invalid injection", field: "payload->>'x') DESC NULLS LAST", valid: false},
		{name: "invalid chars", field: "field-name", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := sanitizeSQLFieldRef(test.field)
			if test.valid && err != nil {
				t.Fatalf("expected valid field ref, got error: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("expected invalid field ref error")
			}
		})
	}
}
