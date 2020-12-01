package sqlChaos

import (
	"strconv"
	"strings"
	"sync"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/api"
	"context"
	"time"
	"delve_tool/log"
	"fmt"
	"errors"
)

/*
		sql hacker用于修改sql 查询操作的返回值
 */
var (
	injectedErr = "\"database/sql/driver\".ErrSkip"
	once = new(sync.Once)
)

//目前只有修改查询的返回值满足要求
var offsetOfFuncs = map[string]int{
	"database/sql.(*DB).QueryContext":   0x48,
	//"database/sql.(*DB).ExecContext":    0x50,
	"database/sql.(*Tx).QueryContext":   0x48,
	//"database/sql.(*Tx).ExecContext":    0x50,
	"database/sql.(*Stmt).QueryContext": 0x38,
	//"database/sql.(*Stmt).ExecContext":  0x40,
}

func Register(funcname string, offset int) {
	offsetOfFuncs[funcname] = offset
}

type sqlHacker struct {
	*rpc2.RPCClient

	breakpoints map[int]string

	errorInfo string
}

func (h *sqlHacker) Invade(ctx context.Context, timeout time.Duration ) error {

	defer func(){
		log.Infof("client disconnecting")
		if _ , err := h.Halt() ; err != nil{
			log.Errorf(" Failed to halt , error - %s", err.Error())
		}
		fmt.Printf("Client Halting....\n")
		if err := h.Disconnect(false) ; err != nil{
			log.Errorf("Failed to disconnect client , error - %s", err.Error())
		}
	}()

	if timeout <= 0 {
		log.Infof("timeout is a negative integer , force it to 10 second")
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
			}else{
				msg += "FunctionReturnLocation return nothing"
			}
			log.Infof("Can't find funcname : %s , error - %s", funcname, msg)
			continue
		}
		haveFunction = true
		log.Infof("Creating breakpoint to funcname : %s", funcname)
		for _, pc := range pcs {
			b, err := h.CreateBreakpoint(&api.Breakpoint{
				Addr: pc,
			})
			if err != nil {
				log.Errorf("CreateBreakpoint error %v", err)
				return err
			}
			h.breakpoints[b.ID] = funcname
		}
	}
	if ! haveFunction{
		log.Errorf("can't find any database/sql function")
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
			log.Errorf("Call expr  %v error %v", expr, err)
		}
	})
}

func (h *sqlHacker) invade(ctx context.Context, period time.Duration) error {
	log.Infof("Start invading for %v", period)
	for {
		select {
		case <-ctx.Done():
			log.Infof("period Context done")
			return nil
		case state, ok := <-h.Continue():
			if !ok {
				return nil
			}
			log.Infof("Continuing...")
			if state.CurrentThread == nil || state.CurrentThread.Breakpoint == nil{
				if state.CurrentThread == nil{
					log.Infof("status current Thread is nil")
				}else{
					log.Infof("status current Thread Breakpoint is nil")
				}
				continue
			}
			id := state.CurrentThread.Breakpoint.ID

			_, err := h.Step()
			if err != nil {
				log.Errorf("Step error %v", err)
				return err
			}

			regs, err := h.ListScopeRegisters(api.EvalScope{
				GoroutineID: -1,
			}, true)
			if err != nil {
				log.Errorf("ListScopeRegisters error %v", err)
				return err
			}

			for _, reg := range regs {
				// Rsp 寄存器的编号是7
				if reg.DwarfNumber == 7 {
					value, err := strconv.ParseInt(strings.Trim(reg.Value, "\""), 0, 64)
					if err != nil {
						log.Errorf("ParseInt %v  error %v", reg.Value, err)
						return err
					}
					offset := offsetOfFuncs[h.breakpoints[id]]
					expr := fmt.Sprintf("*(*error)(%d) = %s", int(value)+offset, injectedErr)
					_, err = h.Call(state.CurrentThread.GoroutineID, expr, false)
					if err != nil {
						log.Errorf("Call expr  %v error %v", expr, err)
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


func newSqlHacker(c *rpc2.RPCClient , errorInfo string) *sqlHacker {
	log.Infof("New sql hacker....")
	hacker := &sqlHacker{
		RPCClient: c,
		breakpoints: make(map[int]string),
		errorInfo: errorInfo,
	}
	return hacker
}

