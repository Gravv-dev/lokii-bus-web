package store

import "testing"

func TestPasswordsAreHashedAndVerified(t *testing.T) {
	password := "cookies-are-not-plaintext-42"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}
	if string(hash) == password {
		t.Fatal("stored password hash must not equal plaintext")
	}
	if !passwordMatches(hash, []byte(password)) {
		t.Fatal("passwordMatches() rejected the correct password")
	}
	if passwordMatches(hash, []byte("another-password")) {
		t.Fatal("passwordMatches() accepted an incorrect password")
	}
}

func TestOrderStatusTransitions(t *testing.T) {
	tests := []struct {
		current string
		next    string
		want    bool
	}{
		{"pending", "processing", true},
		{"pending", "cancelled", true},
		{"processing", "shipped", true},
		{"shipped", "completed", true},
		{"cancelled", "pending", false},
		{"completed", "cancelled", false},
		{"pending", "shipped", false},
	}
	for _, test := range tests {
		if got := validStatusTransition(test.current, test.next); got != test.want {
			t.Errorf("validStatusTransition(%q, %q) = %v, want %v", test.current, test.next, got, test.want)
		}
	}
}
