# testexec

[![Go Report Card](https://goreportcard.com/badge/github.com/brian14708/testexec)](https://goreportcard.com/report/github.com/brian14708/testexec)
[![Go Reference](https://pkg.go.dev/badge/github.com/brian14708/testexec.svg)](https://pkg.go.dev/github.com/brian14708/testexec)

Run subprocess tests for Go.

## How to use

Setup code:

```go
func TestMain(m *testing.M) {
	testexec.Main(m)
}
```

Add subprocess program:

```go
var flagProgram = testexec.NewProgram(func(t *testexec.T, in int, out *int) {
	var f = flag.Int("data", 1, "")
	flag.Parse()
	*out = in + *f
})
```

Add test:

```go
func TestFlag(t *testing.T) {
	var i int
	testexec.Run(t, flagProgram, 12, &i, "--data=123")
	assert.Equal(t, 135, i)
}
```
