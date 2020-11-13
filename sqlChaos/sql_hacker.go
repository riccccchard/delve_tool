package sqlChaos

import (
	"context"
	"errors"
	"fmt"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
	需要注入异常的sql函数的error变量的偏移量
*/
var (
	injectedErr = "\"database/sql/driver\".ErrSkip"
	once = new(sync.Once)
)

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

type sqlHacker struct {
	*rpc2.RPCClient

	breakpoints map[int]string

	errorInfo string
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

//修改error的string field，只需一次
func (h *sqlHacker) modifyErrorStringField (strptr int , goroutineId int) {
	once.Do(func (){
		expr := fmt.Sprintf("*(*string)(*(*int)(%d)) = \"%s\"", strptr , h.errorInfo)
		_ , err := h.Call(goroutineId, expr, false)
		if err != nil{
			log.Errorf("[sqlHacker.invade]Call expr  %v error %v", expr, err)
		}
	})
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
			if state.CurrentThread == nil{
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
					_, err = h.Call(state.CurrentThread.GoroutineID, expr, false)
					if err != nil {
						log.Errorf("[sqlHacker.invade]Call expr  %v error %v", expr, err)
						return err
					}
					if h.errorInfo != ""{
						h.modifyErrorStringField(int(value)+offset+8, state.CurrentThread.GoroutineID)
					}
				}
			}
		}
	}
}


func NewSqlHacker(c *rpc2.RPCClient , errorInfo string) *sqlHacker {
	log.Infof("[NewSqlHacker]New sql hacker....")
	hacker := &sqlHacker{
		RPCClient: c,
		breakpoints: make(map[int]string),
		errorInfo: errorInfo,
	}
	return hacker
}

