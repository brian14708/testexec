package testexec

import (
	"fmt"
	"os"
)

type T struct {
	name string
}

func (t *T) Errorf(format string, args ...interface{}) {
	if t.name == "" {
		fmt.Print("[subprocess] ")
	} else {
		fmt.Printf("[subprocess %s] ", t.name)
	}
	fmt.Printf(format, args...)
	fmt.Print("\n")
	os.Exit(1)
}
