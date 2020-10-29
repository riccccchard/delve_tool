package delveClient

import (
	"fmt"
	"github.com/go-delve/delve/service/api"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"strconv"
	"strings"
	"time"
)

/*
		修改db.Query的error返回值
 */

const(
	//函数名
	sql_query_function_name = "database/sql.(*DB).Query"
	//error在栈中的偏移
	sql_error_offset        = 0x48
	//我们要执行的expression
	expr                    = "*(*error)(%d) = \"database/sql/driver\".ErrBadConn"
)

//在waitTime时间内，修改所有db.Query的返回值
func (dc *DelveClient) setSqlQueryError (workTime time.Duration) error {
	log.Infof("[DelveClient.SetMysqlQueryError]start set sql query error....")
	locs , err := dc.client.FindLocation(api.EvalScope{GoroutineID: -1}, sql_query_function_name , true)
	if err != nil{
		log.Errorf("[DelveClient.SetMysqlQueryError]Failed to find locations , error - %s", err.Error())
		return err
	}

	instructions , err := dc.client.DisassemblePC(api.EvalScope{GoroutineID: -1}, locs[0].PC, api.GNUFlavour)
	if err != nil{
		msg := fmt.Sprintf("[DelveClient.SetMysqlQueryError]Failed to disassemble pc , error - %s", err.Error())
		log.Errorf(msg)
		return err
	}
	var pos int
	for i , inst := range instructions{
		if inst.DestLoc == nil {
			continue
		}
		if inst.DestLoc.Function == nil {
			continue
		}
		if strings.Contains(inst.DestLoc.Function.Name(), "QueryContext") {
			pos = i
			break
		}
	}

	b0, err := dc.client.CreateBreakpoint(&api.Breakpoint{
		Addr: instructions[pos+1].Loc.PC,
	})
	if err != nil {
		msg := fmt.Sprintf("[DelveClient.SetSqlQueryError]Failed to create break point , error - %s", err.Error())
		log.Errorf(msg)
		return err
	}
	log.Infof("[DelveClient.SetSqlQueryError]Create breakpoint at File&Line : %s:%d", b0.File, b0.Line)
	//修改这么多秒
	timeTicker := time.NewTicker(workTime)
	for{
		select {
		case <- timeTicker.C:{
			log.Infof("[DelveClient.SetSqlQueryError]Time is over , close client ....")
			goto timedone
		}
		default:
			_ = <- dc.client.Continue()
			regs, err := dc.client.ListScopeRegisters(api.EvalScope{
				GoroutineID: -1,
			}, true)

			if err != nil {
				msg := fmt.Sprintf("Failed to List Scope Register , error - %s", err.Error())
				log.Errorf("[DelveClient.SetSqlQueryError]%s", msg)
				return err
			}
			for _, reg := range regs {
				if reg.Name == "Rsp" {
					value, err := strconv.ParseInt(strings.Trim(reg.Value, "\""), 0, 64)
					if err != nil {
						msg := fmt.Sprintf("Failed to convert value , error - %s", err.Error())
						log.Errorf("[DelveClient.SetSqlQueryError]%s", msg)
						return err
					}
					expression := fmt.Sprintf(expr, value+sql_error_offset)
					_ , err = dc.client.Call(-1, expression, false)
					if err != nil {
						msg := fmt.Sprintf("Failed to call expr, error - %s", err.Error())
						log.Errorf("[DelveClient.SetSqlQueryError]%s", msg)
						return err
					}
				}
			}
		}
	}
timedone:
	dc.client.Disconnect(true)
	log.Infof("[DelveClient.SetSqlQueryError]client disconnect....")
	return nil
}
