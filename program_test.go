package testexec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	Main(m)
}

func TestEmbeddedProgram(t *testing.T) {
	assert.Panics(t, func() {
		NewProgram(func(*T) {})
	})
}

var repReqPrg = NewProgram(func(t *T, in *int, out *int) {
	*out = *in * 2
})

func TestRepReq(t *testing.T) {
	var i int
	Run(t, repReqPrg, 12, &i)
	assert.Equal(t, 24, i)
}

var chanPrg = NewProgram(func(t *T, in <-chan int, out chan<- int) {
	for i := range in {
		out <- i * 2
	}
})

func TestChan(t *testing.T) {
	i := make(chan int, 2)
	o := make(chan int, 2)
	go Run(t, chanPrg, i, o)

	i <- 1
	assert.Equal(t, 2, <-o)
	i <- 4
	assert.Equal(t, 8, <-o)
	close(i)
	_, ok := <-o
	assert.False(t, ok)
}

func TestChanNil(t *testing.T) {
	o := make(chan int)
	go Run(t, chanPrg, nil, o)
	_, ok := <-o
	assert.False(t, ok)
}

var badType = NewProgram(func(t *T, in <-chan func() int, out chan<- int) {
	for i := range in {
		out <- i() * 2
	}
})

func TestBadType(t *testing.T) {
	assert.Panics(t, func() {
		i := make(chan func() int, 1)
		o := make(chan int)
		i <- func() int {
			return 1
		}
		Run(t, badType, i, o)
	})
}
