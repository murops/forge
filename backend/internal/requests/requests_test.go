package requests

import (
	"errors"
	"strings"
	"testing"
)

func TestCreateInputValidate(t *testing.T) {
	tests := []struct {
		name  string
		in    CreateInput
		field string
	}{
		{"valid", CreateInput{Name: "payments-api", Template: "api-go"}, ""},
		{"digits only", CreateInput{Name: "123", Template: "api-go"}, ""},
		{"too short", CreateInput{Name: "ab", Template: "api-go"}, "name"},
		{"too long", CreateInput{Name: strings.Repeat("a", 101), Template: "api-go"}, "name"},
		{"uppercase", CreateInput{Name: "Payments", Template: "api-go"}, "name"},
		{"leading hyphen", CreateInput{Name: "-payments", Template: "api-go"}, "name"},
		{"trailing hyphen", CreateInput{Name: "payments-", Template: "api-go"}, "name"},
		{"underscore", CreateInput{Name: "pay_ments", Template: "api-go"}, "name"},
		{"long description", CreateInput{Name: "payments", Template: "api-go", Description: strings.Repeat("x", 351)}, "description"},
		{"missing template", CreateInput{Name: "payments"}, "template"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Validate()
			if tt.field == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if verr.Field != tt.field {
				t.Errorf("field = %q, want %q", verr.Field, tt.field)
			}
		})
	}
}
