package testexec

import (
	"context"
	"syscall"
	"testing"
)

func Run(t testing.TB, prg ProgramHandle, req, resp interface{}, args ...string) {
	err := CommandContext(t, prg, nil, req, resp, args...).Run()
	if err != nil {
		t.Errorf("program execute failed: %s", err)
	}
}

func Command(t testing.TB, prg ProgramHandle, req, resp interface{}, args ...string) *Cmd {
	return CommandContext(t, prg, nil, req, resp, args...)
}

func CommandContext(t testing.TB, prg ProgramHandle, ctx context.Context, req, resp interface{}, args ...string) *Cmd {
	if !calledFromMain {
		panic("to use testexec.Main must be called")
	}

	cmd := prg.exec(ctx, req, resp, args)

	t.Cleanup(func() {
		if cmd.Cmd.Process != nil {
			_ = cmd.Cmd.Process.Signal(syscall.SIGTERM)
		}
		_ = cmd.Cmd.Wait()
	})
	return cmd
}
