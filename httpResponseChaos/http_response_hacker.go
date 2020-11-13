package httpResponseChaos

import (
	"context"
	"errors"
	"fmt"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"time"
)

var offsetOfFuncs = map[string]int{
	"net/http.(*response).WriteHeader" : 0,
}

type httpResponseHacker struct {
	*rpc2.RPCClient

	breakpoints map[int]string

	statusCode  int
}



func (h *httpResponseHacker) Invade(ctx context.Context, timeout time.Duration) error {
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

func (h *httpResponseHacker) createBreakpoints(ctx context.Context) error {
	haveFunction := false
	for funcname := range offsetOfFuncs {
		pcs, err := h.FunctionReturnLocations(funcname)
		if err != nil || len(pcs) == 0 {
			msg := ""
			if err != nil{
				msg += err.Error()
			}
			log.Infof("[httpResponseHacker.createBreakpoints] Can't find funcname : %s , error - %s", funcname, msg)
			continue
		}
		haveFunction = true
		log.Infof("[httpResponseHacker.createBreakpoints] Creating breakpoint to funcname : %s", funcname)
		for _, pc := range pcs {
			b, err := h.CreateBreakpoint(&api.Breakpoint{
				Addr: pc,
			})
			if err != nil {
				log.Errorf("[httpResponseHacker.createBreakpoints]CreateBreakpoint error %v", err)
				return err
			}
			h.breakpoints[b.ID] = funcname
		}
	}
	if ! haveFunction{
		log.Errorf("[httpResponseHacker.createBreakpoints]can't find any net/http/server.(*response) function")
		return errors.New("can't find any net/http/server.(*response) function")
	}
	return nil
}

func (h *httpResponseHacker) invade(ctx context.Context, period time.Duration) error {
	log.Infof("[httpResponseHacker.invade]Start invading for %v", period)
	for {
		select {
		case <-ctx.Done():
			log.Infof("[httpResponseHacker.invade]period Context done")
			return nil
		case state, ok := <-h.Continue():
			if !ok {
				return nil
			}
			log.Infof("[httpResponseHacker.invade]Continuing...")
			if state.CurrentThread == nil || state.CurrentThread.Breakpoint == nil {
				continue
			}

			expr := fmt.Sprintf("w.status=%d", h.statusCode)
			_ , err := h.Call(-1 , expr , false)
			if err != nil{
				log.Errorf("[httpResponseHacker.invade]Failed to call expr %s , error - %s", expr , err.Error())
				return err
			}
		}
	}
}
func NewHttpResponseHacker(c *rpc2.RPCClient , statusCode int) *httpResponseHacker {
	log.Infof("[NewHttpResponseHacker]New HttpResponse  hacker....")
	hacker := &httpResponseHacker{
		RPCClient: c,
		breakpoints: make(map[int]string),
		statusCode: statusCode,
	}
	return hacker
}
