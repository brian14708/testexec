// +build !go1.14

package testexec

import "testing"

func cleanup(t testing.TB, fn func()) {}
