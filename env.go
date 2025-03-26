package testexec

import (
	"context"
	"syscall"
	"testing"
)

func Run[In, Out any](t testing.TB, prg ProgramHandle[In, Out], req In, resp Out, args ...string) {
	err := CommandContext(t, prg, nil, req, resp, args...).Run()
	if err != nil {
		t.Errorf("program execute failed: %s", err)
	}
}

func Command[In, Out any](t testing.TB, prg ProgramHandle[In, Out], req In, resp Out, args ...string) *Cmd {
	return CommandContext(t, prg, nil, req, resp, args...)
}

func CommandContext[In, Out any](t testing.TB, prg ProgramHandle[In, Out], ctx context.Context, req In, resp Out, args ...string) *Cmd {
	if !calledFromMain {
		panic("to use testexec.Main must be called")
	}

	cmd := prg.exec(ctx, req, resp, args)

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
		_ = cmd.Cmd.Wait()
	})
	return cmd
}
