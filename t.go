package testexec

import (
	"fmt"
	"os"
)

type T struct {
	Name string
}

func (t *T) Errorf(format string, args ...interface{}) {
	if t.Name == "" {
		fmt.Print("[subprocess] ")
	} else {
		fmt.Printf("[subprocess %s] ", t.Name)
	}
	fmt.Printf(format, args...)
	fmt.Print("\n")
	os.Exit(1)
}
