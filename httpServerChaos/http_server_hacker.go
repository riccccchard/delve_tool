package httpServerChaos

import (
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"strconv"
	"strings"
	"time"
	"context"
	"delve_tool/log"
	"errors"
	"fmt"
)

/*
		将http server read request hacked 掉，将返回400错误
 */

var injectedErr = "ErrBodyNotAllowed"

var offsetOfHackerFuncs = map[string]int{
	"net/http.(*conn).readRequest" : 0x20,
}

type httpServerRequestHacker struct {
	*rpc2.RPCClient

	breakpoints map[int]string
}



func (h *httpServerRequestHacker) Invade(ctx context.Context, timeout time.Duration) error {
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

func (h *httpServerRequestHacker) createBreakpoints(ctx context.Context) error {
	haveFunction := false
	for funcname := range offsetOfHackerFuncs {
		pcs, err := h.FunctionReturnLocations(funcname)
		if err != nil || len(pcs) == 0 {
			msg := ""
			if err != nil{
				msg += err.Error()
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
				log.Errorf(" CreateBreakpoint error %v", err)
				return err
			}
			h.breakpoints[b.ID] = funcname
		}
	}
	if ! haveFunction{
		log.Errorf("can't find any net/http/server.go function")
		return errors.New("can't find any net/http/server.go function")
	}
	return nil
}

func (h *httpServerRequestHacker) invade(ctx context.Context, period time.Duration) error {
	log.Infof(" Start invading for %v", period)
	for {
		select {
		case <-ctx.Done():
			log.Infof(" period Context done")
			return nil
		case state, ok := <-h.Continue():
			if !ok {
				log.Errorf("continuing failed")
				return nil
			}
			log.Infof(" Continuing...")
			if state.CurrentThread == nil || state.CurrentThread.Breakpoint == nil {
				continue
			}
			id := state.CurrentThread.Breakpoint.ID

			_, err := h.Step()
			if err != nil {
				log.Errorf(" Step error %v", err)
				return err
			}

			regs, err := h.ListScopeRegisters(api.EvalScope{
				GoroutineID: -1,
			}, true)
			if err != nil {
				log.Errorf(" ListScopeRegisters error %v", err)
				return err
			}

			for _, reg := range regs {
				// Rsp 寄存器的编号是7
				if reg.DwarfNumber == 7 {
					value, err := strconv.ParseInt(strings.Trim(reg.Value, "\""), 0, 64)
					if err != nil {
						log.Errorf(" ParseInt %v  error %v", reg.Value, err)
						return err
					}
					offset := offsetOfHackerFuncs[h.breakpoints[id]]
					expr := fmt.Sprintf("*(*error)(%d) = %s", int(value)+offset, injectedErr)
					_, err = h.Call(-1, expr, false)
					if err != nil {
						log.Errorf(" Call expr  %v error %v", expr, err)
						return err
					}
				}
			}
		}
	}
}

func newHttpRequestHacker(c *rpc2.RPCClient ) *httpServerRequestHacker {
	log.Infof(" New HttpRequest hacker....")
	hacker := &httpServerRequestHacker{
		RPCClient: c,
		breakpoints: make(map[int]string),
	}
	return hacker
}
