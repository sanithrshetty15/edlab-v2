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
