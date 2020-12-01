package httpServerChaos

import (
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"errors"
	"time"
	"delve_tool/log"
	"fmt"
	"context"
)

var offsetOfDelayFuncs = []string{
	"net/http.(*conn).readRequest",
}


type httpServerDelayer struct{
	*rpc2.RPCClient

	breakpoints map[int]string

	delay time.Duration

}


func (h *httpServerDelayer) Invade(ctx context.Context, timeout time.Duration) error {
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

	if h.delay > timeout{
		log.Infof("delay can't large than period , force it to period")
		h.delay = timeout
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

func (h *httpServerDelayer) createBreakpoints(ctx context.Context) error{
	haveFunction := false
	for _ , funcname := range offsetOfDelayFuncs {
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

func (h *httpServerDelayer) invade (ctx context.Context , period time.Duration) error{
	log.Infof("Start invade for period %v", period)

	for{
		select{
		case <- ctx.Done():
			log.Infof("period context done.")
			return nil
		case state , ok := <- h.Continue():
			if !ok {
				log.Errorf("continuing failed")
				return nil
			}

			log.Infof("Continuing")

			if state.CurrentThread == nil  {
				continue
			}


			goroutineId := -1
			if state.CurrentThread.GoroutineID != 0{
				goroutineId = state.CurrentThread.GoroutineID
			}

			udelay := h.delay.Microseconds()
			if udelay <= 0 {
				log.Infof("delay time too long , force it to 1 second")
				udelay = 1000000
			}
			//runtime.usleep 传入单位为微秒
			expr := fmt.Sprintf("runtime.usleep(%v)", udelay)
			_ , err := h.Call(goroutineId , expr , false )

			if err != nil{
				log.Errorf("call %s error - %s", expr , err.Error())
				return err
			}
		}
	}
}

func newHttpServerDelayChaos (c *rpc2.RPCClient , delay time.Duration) *httpServerDelayer {
	log.Infof("new http server delay")
	hacker := &httpServerDelayer{
		RPCClient: c,
		breakpoints: make(map[int]string),
		delay: delay,
	}
	return hacker
}
