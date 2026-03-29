package vm_test

import (
	"strings"
	"testing"
)

func TestChatroomSelection(t *testing.T) {
	r := runCBAT(t, `.header
    filename "test.bat"
.instrs
    stp ROOM,"Room:"
    ieq ROOM,"1",CR1
    ieq ROOM,"2",CR2
    e "invalid"
    trm
    l CR1
    e "Room 1 selected"
    trm
    l CR2
    e "Room 2 selected"
    trm`, "1")

	t.Logf("Output: %q", r.stdout)
	t.Logf("Variables: %v", r.state["variables"])

	if !strings.Contains(r.stdout, "Room 1 selected") {
		t.Errorf("expected 'Room 1 selected', got: %s", r.stdout)
	}
}
