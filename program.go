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

type Nil chan struct{ private struct{} }

var (
	calledFromMain = false
	programs       = make(map[string]program)
	nilChanType    = typeOf[Nil]()
)

const envKey = "TESTEXEC_KEY"

type ProgramHandle[In any, Out any] string

type program struct {
	fn      func(*T, io.Reader, io.WriteCloser)
	options options
}

func NewProgram[In any, Out any](f func(*T, In, Out), opts ...ProgramOption) ProgramHandle[In, Out] {
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

	inType := typeOf[In]()
	if inType.Kind() == reflect.Chan {
		if inType == nilChanType {
			// noop
		} else if inType.ChanDir() != reflect.RecvDir {
			panic("request parameter must be recv channel")
		}
	}

	outType := typeOf[Out]()
	if outType.Kind() == reflect.Chan {
		if outType == nilChanType {
			// noop
		} else if outType.ChanDir() != reflect.SendDir {
			panic("response parameter must be send channel")
		}
	} else if outType.Kind() != reflect.Ptr {
		panic("response parameter must be channel or pointer")
	}

	prg.fn = func(t *T, in io.Reader, out io.WriteCloser) {
		var (
			inArg  In
			outArg Out
		)

		switch inType.Kind() {
		case reflect.Chan:
			ch := makeChan[In](0)
			setValue(&inArg, ch)
			go func() {
				if err := decodeValues(ch, in); err != nil {
					panic(fmt.Sprintf("fail to decode request: %s", err))
				}
			}()
		default:
			if err := decodeSingle(&inArg, in); err != nil {
				panic(fmt.Sprintf("fail to decode request: %s", err))
			}
		}

		switch outType.Kind() {
		case reflect.Chan:
			done := make(chan struct{})
			ch := makeChan[Out](0)
			setValue(&outArg, ch)
			go func() {
				if err := encodeValues(ch, out); err != nil {
					panic(fmt.Sprintf("fail to encode response: %s", err))
				}
				close(done)
			}()

			f(t, inArg, outArg)
			ch.Close()
			<-done

		case reflect.Ptr:
			setValue(&outArg, reflect.New(outType.Elem()))
			f(t, inArg, outArg)
			if err := encodeSingle(outArg, out); err != nil {
				panic(fmt.Sprintf("fail to encode response: %s", err))
			}
		}
	}
	programs[key] = prg

	return ProgramHandle[In, Out](key)
}

func (c ProgramHandle[In, Out]) exec(ctx context.Context, request In, response Out, args []string) *Cmd {
	cmd := &Cmd{ctx: ctx}

	if ctx != nil {
		cmd.Cmd = exec.CommandContext(ctx, os.Args[0], args...)
	} else {
		cmd.Cmd = exec.Command(os.Args[0], args...)
	}

	cmd.Env = append(os.Environ(), envKey+"="+string(c))
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	inR, inW, _ := os.Pipe()
	outR, outW, _ := os.Pipe()
	cmd.ExtraFiles = []*os.File{inR, outW}
	cmd.closers = append(cmd.closers, inR, inW, outR, outW)

	cmd.wg.Go(func() error {
		var err error
		switch typeOf[Out]().Kind() {
		case reflect.Chan:
			err = encodeValues(reflect.ValueOf(request), inW)
		default:
			err = encodeSingle(request, inW)
		}
		if err != nil {
			return fmt.Errorf("fail to encode request: %w", err)
		}
		return nil
	})
	cmd.wg.Go(func() error {
		var err error
		switch typeOf[Out]().Kind() {
		case reflect.Chan:
			err = decodeValues(reflect.ValueOf(response), outR)
		default:
			err = decodeSingle(&response, outR)
		}
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
		defer func() {
			_ = inPipe.Close()
			_ = outPipe.Close()
		}()

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
