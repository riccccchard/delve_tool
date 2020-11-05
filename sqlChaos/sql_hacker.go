package sqlChaos

import (
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"context"
	"strconv"
	"strings"
	"time"
	"fmt"
	"errors"
)

/*
	需要注入异常的sql函数的error变量的偏移量
*/
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

	breakpoints map[int]string
}

func (h *sqlHacker) Invade(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	sctx, cancel := context.WithTimeout(ctx, timeout)

	err := h.createBreakpoints(sctx)
	if err != nil {
		return err
	}

	if err = h.invade(sctx, timeout) ; err != nil{
		return err
	}
	cancel()
	return nil
}

func (h *sqlHacker) createBreakpoints(ctx context.Context) error {
	haveFunction := false
	for funcname := range offsetOfFuncs {
		pcs, err := h.FunctionReturnLocations(funcname)
		if err != nil || len(pcs) == 0 {
			msg := ""
			if err != nil{
				msg += err.Error()
			}
			log.Infof("[sqlHacker.createBreakpoints] Can't find funcname : %s , error - %s", funcname, msg)
			continue
		}
		haveFunction = true
		log.Infof("[sqlHacker.createBreakpoints] Creating breakpoint to funcname : %s", funcname)
		for _, pc := range pcs {
			b, err := h.CreateBreakpoint(&api.Breakpoint{
				Addr: pc,
			})
			if err != nil {
				log.Errorf("[sqlHacker.createBreakpoints]CreateBreakpoint error %v", err)
				return err
			}
			h.breakpoints[b.ID] = funcname
		}
	}
	if ! haveFunction{
		log.Errorf("[sqlHacker.createBreakpoints]can't find any database/sql function")
		return errors.New("can't find any database/sql function")
	}
	return nil
}

func (h *sqlHacker) invade(ctx context.Context, period time.Duration) error {
	log.Infof("[sqlHacker.invade]Start invading for %v", period)
	for {
		select {
		case <-ctx.Done():
			log.Infof("[sqlHacker.invade]period Context done")
			return nil
		case state, ok := <-h.Continue():
			if !ok {
				return nil
			}
			log.Infof("[sqlHacker.invade]Continuing...")
			if state.CurrentThread == nil || state.CurrentThread.Breakpoint == nil {
				continue
			}
			id := state.CurrentThread.Breakpoint.ID

			_, err := h.Step()
			if err != nil {
				log.Errorf("[sqlHacker.invade]Step error %v", err)
				return err
			}

			regs, err := h.ListScopeRegisters(api.EvalScope{
				GoroutineID: -1,
			}, true)
			if err != nil {
				log.Errorf("[sqlHacker.invade]ListScopeRegisters error %v", err)
				return err
			}

			for _, reg := range regs {
				// Rsp 寄存器的编号是7
				if reg.DwarfNumber == 7 {
					value, err := strconv.ParseInt(strings.Trim(reg.Value, "\""), 0, 64)
					if err != nil {
						log.Errorf("[sqlHacker.invade]ParseInt %v  error %v", reg.Value, err)
						return err
					}
					offset := offsetOfFuncs[h.breakpoints[id]]
					expr := fmt.Sprintf("*(*error)(%d) = %s", int(value)+offset, injectedErr)
					_, err = h.Call(-1, expr, false)
					if err != nil {
						log.Errorf("[sqlHacker.invade]Call expr  %v error %v", expr, err)
						return err
					}
				}
			}
		}
	}
}


func NewSqlHacker(c *rpc2.RPCClient) Hacker {
	log.Infof("[NewSqlHacker]New sql hacker....")
	hacker := &sqlHacker{
		RPCClient: c,
		breakpoints: make(map[int]string),
	}
	return hacker
}

