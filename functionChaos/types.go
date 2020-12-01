package functionChaos

import (
	"delve_tool/types"
	"errors"
	"github.com/go-delve/delve/service/rpc2"
	"time"
)
const (
	Delay_type = "delay"
	Panic_type = "panic"
)


func NewFunctionChaos(r *rpc2.RPCClient , chaosType string , param ...interface{}) (types.ChaosInterface , error){
	switch chaosType {
	case Delay_type:
		var delay time.Duration
		funcs := make([]string , 0 )

		if len(param) <= 1 {
			return nil , errors.New("please input funcnames as parameter")
		}
		if d , ok := param[0].(time.Duration); ok {
			delay = d
		}
		if delay == 0 {
			delay = 1 * time.Second
		}
		str := ""
		for i := 1 ; i < len(param) ; i ++ {
			ok := true
			if str, ok = param[i].(string) ; ! ok {
				return nil, errors.New("input function name is not string")
			}
			funcs = append(funcs , str)
		}
		return newFunctionDelayer(r , delay , funcs) , nil
	case Panic_type:
		funcs := make([]string , 0)
		if len(param) == 0 {
			return nil , errors.New("please input function name as parameter")
		}
		str := ""
		for i := 0 ; i < len(param) ; i ++ {
			ok :=true
			if str, ok = param[i].(string) ; !ok {
				return nil, errors.New("input function name is not string")
			}
			funcs = append(funcs , str)
		}
		return newFunctionPanicer(r, funcs) , nil
	default:
		return nil , errors.New("unknown function chaos type")
	}
}

