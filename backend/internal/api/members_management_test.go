package api

import "testing"

func TestIsGmailAddress(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"member@gmail.com", true},
		{"Member.Name+lab@Gmail.com", true},
		{"member@googlemail.com", false},
		{"member@yahoo.com", false},
		{"member@gmail", false},
		{"@gmail.com", false},
		{"member @gmail.com", false},
	}
	for _, test := range tests {
		if got := isGmailAddress(test.email); got != test.want {
			t.Errorf("isGmailAddress(%q) = %v, want %v", test.email, got, test.want)
		}
	}
}

func TestNormalizeMemberIdentifier(t *testing.T) {
	tests := []struct {
		name           string
		identifierType string
		usn            string
		roll           string
		wantType       string
		wantUSN        string
		wantRoll       string
		wantError      bool
	}{
		{"explicit USN", "usn", "4al25cs001", "", "usn", "4AL25CS001", "", false},
		{"explicit roll number", "roll_number", "", "R123", "roll_number", "", "R123", false},
		{"legacy USN", "", "4AL25CS002", "", "usn", "4AL25CS002", "", false},
		{"legacy roll number", "", "", "R124", "roll_number", "", "R124", false},
		{"missing identifier", "", "", "", "", "", "", true},
		{"both identifiers without type", "", "4AL25CS003", "R125", "", "4AL25CS003", "R125", true},
		{"USN type with roll number", "usn", "4AL25CS004", "R126", "usn", "4AL25CS004", "R126", true},
		{"roll type with USN", "roll_number", "4AL25CS005", "R127", "roll_number", "4AL25CS005", "R127", true},
		{"USN type without value", "usn", "", "", "usn", "", "", true},
		{"roll type without value", "roll_number", "", "", "roll_number", "", "", true},
		{"unknown identifier type", "email", "4AL25CS006", "", "email", "4AL25CS006", "", true},
		{"invalid USN format", "usn", "ABC", "", "usn", "ABC", "", true},
		{"invalid roll format", "roll_number", "", "R 128", "roll_number", "", "R 128", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotType, gotUSN, gotRoll, errs := normalizeMemberIdentifier(
				test.identifierType,
				test.usn,
				test.roll,
			)
			if gotType != test.wantType || gotUSN != test.wantUSN || gotRoll != test.wantRoll {
				t.Fatalf(
					"normalizeMemberIdentifier(%q, %q, %q) = (%q, %q, %q), want (%q, %q, %q)",
					test.identifierType,
					test.usn,
					test.roll,
					gotType,
					gotUSN,
					gotRoll,
					test.wantType,
					test.wantUSN,
					test.wantRoll,
				)
			}
			if gotError := len(errs) > 0; gotError != test.wantError {
				t.Fatalf("normalizeMemberIdentifier(%q, %q, %q) errors = %v, wantError %v",
					test.identifierType, test.usn, test.roll, errs, test.wantError)
			}
		})
	}
}
