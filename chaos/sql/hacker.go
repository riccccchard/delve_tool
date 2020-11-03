package sql

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-delve/delve/pkg/logflags"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
	"github.com/sirupsen/logrus"
)

var injectedErr = "\"database/sql/driver\".ErrBadConn"

var offsetOfFuncs = map[string]int{
	"database/sql.(*DB).QueryContext":   0x48,
	"database/sql.(*DB).ExecContext":    0x50,
	"database/sql.(*Tx).QueryContext":   0x48,
	"database/sql.(*Tx).ExecContext":    0x50,
	"database/sql.(*Stmt).QueryContext": 0x38,
	"database/sql.(*Stmt).ExecContext":  0x40,
}

func Register(funcname string, offset int) {
	offsetOfFuncs[funcname] = offset
}

type Hacker interface {
	Invade(ctx context.Context, period time.Duration) error
}

type sqlHacker struct {
	*rpc2.RPCClient
	*rpccommon.ServerImpl

	breakpoints map[int]string
	log         *logrus.Entry
}

func NewSQLHacker(pid int) (Hacker, error) {
	addr := ":0"
	deb := debugger.Config{
		Backend:   "default",
		AttachPid: pid,
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	hacker := &sqlHacker{
		breakpoints: make(map[int]string),
		log:         logflags.DebuggerLogger(),
	}
	hacker.log.Infof("Serving at %v", lis.Addr())
	server := rpccommon.NewServer(&service.Config{
		Debugger:    deb,
		Listener:    lis,
		AcceptMulti: true,
		APIVersion:  2,
	})
	if err := server.Run(); err != nil {
		return nil, err
	}
	hacker.ServerImpl = server
	hacker.RPCClient = rpc2.NewClient(lis.Addr().String())
	hacker.SetReturnValuesLoadConfig(&api.LoadConfig{})
	return hacker, nil
}

func (h *sqlHacker) Invade(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	sctx, cancel := context.WithTimeout(ctx, timeout)
	defer func() {
		cancel()

		h.log.Info("Disconnect ...")
		if err := h.Disconnect(false); err != nil {
			h.log.Errorf("Disconnect error %v", err)
		}

		// FIXME: the server will be locked in the Continue command and cannot be shutted down.
		h.log.Info("Stoping server ...")
		if err := h.Stop(); err != nil {
			h.log.Errorf("Stop server error %v", err)
		}
	}()

	err := h.createBreakpoints(sctx)
	if err != nil {
		return err
	}

	return h.invade(sctx, timeout)
}

func (h *sqlHacker) createBreakpoints(ctx context.Context) error {
	for funcname := range offsetOfFuncs {
		pcs, err := h.FunctionReturnLocations(funcname)
		if err != nil || len(pcs) == 0 {
			h.log.Errorf("FunctionReturnLocations error %v", err)
			return err
		}

		for _, pc := range pcs {
			b, err := h.CreateBreakpoint(&api.Breakpoint{
				Addr: pc,
			})
			if err != nil {
				h.log.Errorf("CreateBreakpoint error %v", err)
				return err
			}
			h.breakpoints[b.ID] = funcname
		}
	}
	return nil
}

func (h *sqlHacker) invade(ctx context.Context, period time.Duration) error {
	h.log.Infof("Start invading for %v", period)
	for {
		select {
		case <-ctx.Done():
			h.log.Infof("Context done")
			return ctx.Err()
		case state, ok := <-h.Continue():
			if !ok {
				return nil
			}
			h.log.Infof("Continuing...")
			if state.CurrentThread == nil || state.CurrentThread.Breakpoint == nil {
				continue
			}
			id := state.CurrentThread.Breakpoint.ID

			_, err := h.Step()
			if err != nil {
				h.log.Errorf("Step error %v", err)
				return err
			}

			regs, err := h.ListScopeRegisters(api.EvalScope{
				GoroutineID: -1,
			}, true)
			if err != nil {
				h.log.Errorf("ListScopeRegisters error %v", err)
				return err
			}

			for _, reg := range regs {
				// Rsp 寄存器的编号是7
				if reg.DwarfNumber == 7 {
					value, err := strconv.ParseInt(strings.Trim(reg.Value, "\""), 0, 64)
					if err != nil {
						h.log.Errorf("ParseInt %v  error %v", reg.Value, err)
						return err
					}
					offset := offsetOfFuncs[h.breakpoints[id]]
					expr := fmt.Sprintf("*(*error)(%d) = %s", int(value)+offset, injectedErr)
					_, err = h.Call(-1, expr, false)
					if err != nil {
						h.log.Errorf("Call expr  %v error %v", expr, err)
						return err
					}
				}
			}
		}
	}
}
