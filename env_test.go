package testexec

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

var panicPrg = NewProgram(func(t *T) {
	panic("")
})

func TestBasic(t *testing.T) {
	cmd := Command(t, panicPrg, nil, nil)
	assert.Error(t, cmd.Run())
}

var flagPrg = NewProgram(func(t *T) {
	var f = flag.Int64("my", 1, "")
	flag.Parse()
	assert.Equal(t, *f, int64(123))
})

func TestArgs(t *testing.T) {
	Run(t, flagPrg, nil, nil, "--my=123")
}

var errorPrg = NewProgram(func(t *T) {
	assert.Fail(t, "fail")
}, WithName("error program"))

func TestError(t *testing.T) {
	assert.Error(t, Command(t, errorPrg, nil, nil).Run())
}
