package mapper

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	old := "(321) 277-0753"
	new := "+13212770753"

	if n := mustParsePhoneNumber(old); n != new {
		t.Fail()
	} else {
		fmt.Println(n)
	}
}
