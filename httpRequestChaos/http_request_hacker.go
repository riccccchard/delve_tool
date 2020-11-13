package httpRequestChaos

import (
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"strconv"
	"strings"
	"time"
	"context"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"errors"
	"fmt"
)

var injectedErr = "ErrBodyNotAllowed"

var offsetOfFuncs = map[string]int{
	"net/http.(*conn).readRequest" : 0x20,
}

type httpRequestHacker struct {
	*rpc2.RPCClient

	breakpoints map[int]string
}



func (h *httpRequestHacker) Invade(ctx context.Context, timeout time.Duration) error {
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

func (h *httpRequestHacker) createBreakpoints(ctx context.Context) error {
	haveFunction := false
	for funcname := range offsetOfFuncs {
		pcs, err := h.FunctionReturnLocations(funcname)
		if err != nil || len(pcs) == 0 {
			msg := ""
			if err != nil{
				msg += err.Error()
			}
			log.Infof("[httpRequestHacker.createBreakpoints] Can't find funcname : %s , error - %s", funcname, msg)
			continue
		}
		haveFunction = true
		log.Infof("[httpRequestHacker.createBreakpoints] Creating breakpoint to funcname : %s", funcname)
		for _, pc := range pcs {
			b, err := h.CreateBreakpoint(&api.Breakpoint{
				Addr: pc,
			})
			if err != nil {
				log.Errorf("[httpRequestHacker.createBreakpoints]CreateBreakpoint error %v", err)
				return err
			}
			h.breakpoints[b.ID] = funcname
		}
	}
	if ! haveFunction{
		log.Errorf("[httpRequestHacker.createBreakpoints]can't find any net/http/server.go function")
		return errors.New("can't find any net/http/server.go function")
	}
	return nil
}

func (h *httpRequestHacker) invade(ctx context.Context, period time.Duration) error {
	log.Infof("[httpRequestHacker.invade]Start invading for %v", period)
	for {
		select {
		case <-ctx.Done():
			log.Infof("[httpRequestHacker.invade]period Context done")
			return nil
		case state, ok := <-h.Continue():
			if !ok {
				return nil
			}
			log.Infof("[httpRequestHacker.invade]Continuing...")
			if state.CurrentThread == nil || state.CurrentThread.Breakpoint == nil {
				continue
			}
			id := state.CurrentThread.Breakpoint.ID

			_, err := h.Step()
			if err != nil {
				log.Errorf("[httpRequestHacker.invade]Step error %v", err)
				return err
			}

			regs, err := h.ListScopeRegisters(api.EvalScope{
				GoroutineID: -1,
			}, true)
			if err != nil {
				log.Errorf("[httpRequestHacker.invade]ListScopeRegisters error %v", err)
				return err
			}

			for _, reg := range regs {
				// Rsp 寄存器的编号是7
				if reg.DwarfNumber == 7 {
					value, err := strconv.ParseInt(strings.Trim(reg.Value, "\""), 0, 64)
					if err != nil {
						log.Errorf("[httpRequestHacker.invade]ParseInt %v  error %v", reg.Value, err)
						return err
					}
					offset := offsetOfFuncs[h.breakpoints[id]]
					expr := fmt.Sprintf("*(*error)(%d) = %s", int(value)+offset, injectedErr)
					_, err = h.Call(-1, expr, false)
					if err != nil {
						log.Errorf("[httpRequestHacker.invade]Call expr  %v error %v", expr, err)
						return err
					}
				}
			}
		}
	}
}
func NewHttpRequestHacker(c *rpc2.RPCClient ) *httpRequestHacker {
	log.Infof("[NewHttpRequestHacker]New HttpRequest hacker....")
	hacker := &httpRequestHacker{
		RPCClient: c,
		breakpoints: make(map[int]string),
	}
	return hacker
}
