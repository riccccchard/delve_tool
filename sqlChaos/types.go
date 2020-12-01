package sqlChaos

import (
	"delve_tool/types"
	"errors"
	"github.com/go-delve/delve/service/rpc2"
	"time"
)


const (
	Delay_type       =        "delay"
	Conn_pool_type   =    "conn_pool"
	Query_error_type =  "query_error"    //可能不符合业务场景，先保留
)

//chaosType指定类型，param为对应的chaos object需要的参数
func NewSqlChaos(c *rpc2.RPCClient , chaosType string , param ...interface{}) (types.ChaosInterface , error) {
	switch chaosType {
	case Query_error_type:
		info := ""
		if len(param) != 0 {
			if str , ok := param[0].(string) ; ok {
				info = str
			}
		}
		return newSqlHacker(c , info) , nil
	case Delay_type:
		var delay time.Duration
		if len(param) != 0{
			if t , ok := param[0].(time.Duration); ok{
				delay = t
			}
		}
		if int(delay) == 0{
			delay = 1 * time.Second
		}
		return newSqlDelay(c, delay), nil
	case Conn_pool_type:
		count := -1
		serviceInfo := ""
		if len(param) > 1{
			if c , ok := param[0].(int) ; ok {
				count = c
			}
			if s , ok := param[1].(string) ; ok {
				serviceInfo = s
			}
		}
		if serviceInfo == ""{
			return nil , errors.New("please input mysql service  information")
		}
		if count == -1 {
			count = 100
		}
		return newConnPoolHacker(100 , serviceInfo ) , nil
	}

	return nil , errors.New("unknown sql chaos type")
}

