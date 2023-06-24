package bash

import (
	"fmt"
	"testing"
)

func TestHeader(t *testing.T) {
	var hdr Header
	s := fmt.Sprintf("%s", hdr)
	// s = hex.EncodeToString(hdr[:])
	fmt.Println("sssssss", len(s))
	t.Log(s)
}
