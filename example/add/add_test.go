package add

import "testing"

func TestAdd(t *testing.T) {
	if Add(3, 2) != 5 {
		t.Errorf("3 + 2 != 5")
	}
}
