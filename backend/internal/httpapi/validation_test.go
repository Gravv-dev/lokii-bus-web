package httpapi

import "testing"

func TestUUIDValidation(t *testing.T) {
	valid := []string{
		"d9428888-122b-11e1-b85c-61cd3cbb3210",
		"550e8400-e29b-41d4-a716-446655440000",
	}
	for _, value := range valid {
		if !uuidPattern.MatchString(value) {
			t.Errorf("%q should be accepted as a UUID", value)
		}
	}
	for _, value := range []string{"", "1", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		if uuidPattern.MatchString(value) {
			t.Errorf("%q should be rejected as a UUID", value)
		}
	}
}

func TestSlugValidation(t *testing.T) {
	for _, value := range []string{"classic-chip", "brown-butter2"} {
		if !slugPattern.MatchString(value) {
			t.Errorf("%q should be accepted as a slug", value)
		}
	}
	for _, value := range []string{"", "Classic-Chip", "-bad", "two--hyphens", "bad/slug"} {
		if slugPattern.MatchString(value) {
			t.Errorf("%q should be rejected as a slug", value)
		}
	}
}
