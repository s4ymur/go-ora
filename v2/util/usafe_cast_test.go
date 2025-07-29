package util

import "testing"

func Test_unsafeString(t *testing.T) {
	t.Parallel()

	// arrange
	val := []byte{'a', 'b', 'c', 'd', 'e'}

	// act
	res := unsafeString(&val[0], len(val))

	// assert
	if res != "abcde" {
		t.Errorf("expected 'abcde', got '%s'", res)
	}
}
