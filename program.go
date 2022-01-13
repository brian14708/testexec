package testexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"syscall"
	"testing"
)

var (
	calledFromMain = false
	programs       = make(map[string]program)
)

const envKey = "TESTEXEC_KEY"

type ProgramHandle string

type program struct {
	fn      func(*T, io.Reader, io.WriteCloser)
	options options
}

func NewProgram(f interface{}, opts ...ProgramOption) ProgramHandle {
	if calledFromMain {
		panic("should not call NewProgram after RunTestMain")
	}

	var prg program
	for _, o := range opts {
		o(&prg.options)
	}

	var key string
	if _, file, line, ok := runtime.Caller(1); ok {
		key = fmt.Sprintf("%d|%s", line, file)
	}
	if _, ok := programs[key]; key == "" || ok {
		panic(fmt.Sprintf("invalid exec_key = `%s'", key))
	}

	typ := reflect.TypeOf(f)
	if typ.Kind() != reflect.Func ||
		(typ.NumIn() != 3 && typ.NumIn() != 1) ||
		typ.NumOut() != 0 {
		panic("program must be function in the form of `func(*testexec.T, req, resp)' or `func(*testexec.T)'")
	}

	if typ.NumIn() == 1 {
		prg.fn = func(t *T, _ io.Reader, _ io.WriteCloser) {
			reflect.ValueOf(f).Call([]reflect.Value{
				reflect.ValueOf(t),
			})
		}
	} else {
		inType, outType := typ.In(1), typ.In(2)

		if inType.Kind() == reflect.Chan {
			if inType.ChanDir() != reflect.RecvDir {
				panic("request parameter must be recv channel")
			}
		}

		if outType.Kind() == reflect.Chan {
			if outType.ChanDir() != reflect.SendDir {
				panic("response parameter must be send channel")
			}
		} else if outType.Kind() != reflect.Ptr {
			panic("response parameter must be channel or pointer")
		}

		prg.fn = func(t *T, in io.Reader, out io.WriteCloser) {
			inArg := reflect.New(inType).Elem()
			switch inType.Kind() {
			case reflect.Chan:
				ch := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, inType.Elem()), 0)
				go func() {
					if err := decodeValues(ch, in); err != nil {
						panic(fmt.Sprintf("fail to decode request: %s", err))
					}
				}()

				inArg.Set(ch)
			default:
				if err := decodeValues(inArg, in); err != nil {
					panic(fmt.Sprintf("fail to decode request: %s", err))
				}
			}

			switch outType.Kind() {
			case reflect.Chan:
				done := make(chan struct{})
				ch := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, outType.Elem()), 0)
				go func() {
					if err := encodeValues(ch, out); err != nil {
						panic(fmt.Sprintf("fail to encode response: %s", err))
					}
					close(done)
				}()

				outArg := reflect.New(outType).Elem()
				outArg.Set(ch)

				reflect.ValueOf(f).Call([]reflect.Value{
					reflect.ValueOf(t), inArg, outArg,
				})
				ch.Close()
				<-done

			case reflect.Ptr:
				outArg := reflect.New(outType.Elem())
				reflect.ValueOf(f).Call([]reflect.Value{
					reflect.ValueOf(t), inArg, outArg,
				})
				if err := encodeValues(outArg, out); err != nil {
					panic(fmt.Sprintf("fail to encode response: %s", err))
				}
			}
		}
	}
	programs[key] = prg

	return ProgramHandle(key)
}

func (c ProgramHandle) exec(ctx context.Context, request, response interface{}, args []string) *Cmd {
	cmd := &Cmd{ctx: ctx}

	if ctx != nil {
		cmd.Cmd = exec.CommandContext(ctx, os.Args[0], args...)
	} else {
		cmd.Cmd = exec.Command(os.Args[0], args...)
	}

	cmd.Cmd.Env = append(os.Environ(), envKey+"="+string(c))
	if wd, err := os.Getwd(); err == nil {
		cmd.Cmd.Dir = wd
	}
	cmd.Cmd.Stderr = os.Stderr
	cmd.Cmd.Stdout = os.Stdout
	cmd.Cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	inR, inW, _ := os.Pipe()
	outR, outW, _ := os.Pipe()
	cmd.Cmd.ExtraFiles = []*os.File{inR, outW}
	cmd.closers = append(cmd.closers, inR, inW, outR, outW)

	cmd.wg.Go(func() error {
		err := encodeValues(reflect.ValueOf(request), inW)
		if err != nil {
			return fmt.Errorf("fail to encode request: %w", err)
		}
		return nil
	})
	cmd.wg.Go(func() error {
		err := decodeValues(reflect.ValueOf(response), outR)
		if err != nil {
			return fmt.Errorf("fail to decode response: %w", err)
		}
		return nil
	})

	return cmd
}

func Main(m *testing.M) {
	calledFromMain = true
	if key := os.Getenv(envKey); key != "" {
		inPipe := os.NewFile(uintptr(3), "")
		outPipe := os.NewFile(uintptr(4), "")
		defer inPipe.Close()
		defer outPipe.Close()

		prg, ok := programs[key]
		if !ok {
			panic(fmt.Sprintf("program not found, exec_key = %s", key))
		}

		prg.fn(&T{
			name: prg.options.Name,
		}, inPipe, outPipe)
		return
	}

	os.Exit(m.Run())
}
