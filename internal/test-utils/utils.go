package utils

import (
	"os"
	"strings"
)

var Debug = func() bool {
	return strings.HasSuffix(os.Args[0], "__debug_bin")
}()
