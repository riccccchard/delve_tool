package sqlChaos

import (
	"context"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"errors"
	"time"
	"delve_tool/log"
	"fmt"
)

/*
		sql delay用于对sql操作做延迟，包括sql的增删改
 */

type sqlDelayer struct {
	*rpc2.RPCClient

	breakpoints map[int]string

	delay time.Duration
}

//database/sql的增删改查函数
var sqlDelayFunctions = []string{
	"database/sql.(*DB).QueryContext",
	"database/sql.(*DB).ExecContext",
	"database/sql.(*Tx).QueryContext",
	"database/sql.(*Tx).ExecContext",
	"database/sql.(*Stmt).QueryContext",
	"database/sql.(*Stmt).ExecContext",
}

func (d *sqlDelayer) Invade (ctx context.Context , timeout time.Duration) error {
	defer func(){
		log.Infof("client disconnecting")
		if _ , err := d.Halt() ; err != nil{
			log.Errorf(" Failed to halt , error - %s", err.Error())
		}
		fmt.Printf("Client Halting....\n")
		if err := d.Disconnect(false) ; err != nil{
			log.Errorf("Failed to disconnect client , error - %s", err.Error())
		}
	}()

	if timeout <= 0 {
		log.Infof("timeout is a negative integer , force it to 10 second")
		timeout = 10 * time.Second
	}

	if d.delay > timeout{
		log.Infof("delay can't large than period , force it to period")
		d.delay = timeout
	}

	sctx , cancel := context.WithTimeout(ctx , timeout)

	if err := d.createBreakpoints(sctx); err != nil{
		return err
	}

	if err := d.invade(sctx, timeout) ; err != nil{
		return err
	}
	cancel()
	return nil
}
func (d sqlDelayer) createBreakpoints (ctx context.Context) error{
	haveFunction := false
	for _ , funcname := range sqlDelayFunctions {
		locs, err := d.FindLocation(api.EvalScope{GoroutineID: -1} , funcname , false)
		if err != nil || len(locs) == 0 {
			msg := ""
			if err != nil{
				msg += err.Error()
			}else{
				msg += "findLocation return nothing"
			}
			log.Infof("Can't find funcname : %s , error - %s", funcname, msg)
			continue
		}
		haveFunction = true
		log.Infof("Creating breakpoint to funcname : %s", funcname)
		for _, loc := range locs {
			b, err := d.CreateBreakpoint(&api.Breakpoint{
				Addr: loc.PC,
			})
			if err != nil {
				log.Errorf("CreateBreakpoint error %s", err.Error())
				return err
			}
			d.breakpoints[b.ID] = funcname
		}
	}
	if ! haveFunction{
		log.Errorf("can't find any database/sql function")
		return errors.New("can't find any database/sql function")
	}
	return nil
}
func (d *sqlDelayer) invade (ctx context.Context , period time.Duration ) error{

	log.Infof("start invading for %v , delay %v", period, d.delay)
	for {
		select {
		case <- ctx.Done():
			log.Infof("period Context done.")
			return nil
		case state , ok := <- d.Continue():
			if !ok {
				log.Infof("continue false , quiting")
				return nil
			}
			log.Infof("Continuing")
			if state.CurrentThread == nil{
				log.Infof("status current Thread is nil")
				continue
			}
			if state.CurrentThread.Breakpoint != nil{
				id := state.CurrentThread.Breakpoint.ID
				log.Infof("calling delay for function : %s", sqlDelayFunctions[id])
			}
			goroutineId := -1
			if state.CurrentThread.GoroutineID != 0{
				goroutineId = state.CurrentThread.GoroutineID
			}

			udelay := d.delay.Microseconds()
			if udelay <= 0 {
				log.Infof("delay time too long , force it to 1 second")
				udelay = 1000000
			}
			//runtime.usleep 传入单位为微秒
			expr := fmt.Sprintf("runtime.usleep(%v)", udelay)
			_ , err := d.Call(goroutineId , expr , false )

			if err != nil{
				log.Errorf("call %s error - %s", expr , err.Error())
				return err
			}
		}
	}
}

func newSqlDelay(c *rpc2.RPCClient , delay time.Duration ) *sqlDelayer {
	log.Infof("New sql delay....")
	hacker := &sqlDelayer{
		RPCClient: c,
		delay: delay,
		breakpoints: make(map[int]string),
	}
	return hacker
}



