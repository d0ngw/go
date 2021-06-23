package common

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessFileLines(t *testing.T) {
	f := path.Join("testdata", "1.txt")
	var fp = func(line string, lineNum int, readErr error) (stop bool) {
		assert.NoError(t, readErr)
		line = strings.TrimSpace(line)
		fmt.Println(lineNum, line)
		return
	}
	ProcessFileLines(f, fp)
}
