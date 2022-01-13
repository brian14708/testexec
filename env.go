package testexec

import (
	"context"
	"syscall"
	"testing"
)

func Run(t testing.TB, prg Program, req, resp interface{}, args ...string) {
	err := CommandContext(t, prg, nil, req, resp, args...).Run()
	if err != nil {
		t.Errorf("program execute failed: %s", err)
	}
}

func Command(t testing.TB, prg Program, req, resp interface{}, args ...string) *Cmd {
	return CommandContext(t, prg, nil, req, resp, args...)
}

func CommandContext(t testing.TB, prg Program, ctx context.Context, req, resp interface{}, args ...string) *Cmd {
	if !calledFromMain {
		panic("to use testexec RunTestMain must be called")
	}

	cmd := prg.exec(ctx, req, resp, args)

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
		_ = cmd.Wait()
	})
	return cmd
}
