package testexec

import (
	"context"
	"io"
	"os/exec"

	"golang.org/x/sync/errgroup"
)

type Cmd struct {
	*exec.Cmd
	ctx     context.Context
	wg      errgroup.Group
	closers []io.Closer
}

func (c *Cmd) Wait() error {
	err := c.Cmd.Wait()
	for _, c := range c.closers {
		c.Close()
	}
	c.closers = nil

	if wgErr := c.wg.Wait(); wgErr != nil {
		c.wg = errgroup.Group{}
		panic(wgErr)
	}
	if c.ctx != nil {
		if err := c.ctx.Err(); err != nil {
			return err
		}
	}
	return err
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}
