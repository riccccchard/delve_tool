package httpServerChaos

import (
	"delve_tool/types"
	"errors"
	"github.com/go-delve/delve/service/rpc2"
	"time"
)


const(
	Delay_type         = "delay"
	Request_error_type = "request_error"
)

func NewHttpServerHacker(r *rpc2.RPCClient , chaosType string , param ...interface{}) (types.ChaosInterface, error) {
	switch chaosType {
	case Delay_type:
		var delay time.Duration
		if len(param) != 0 {
			if d , ok := param[0].(time.Duration) ; ok {
				delay = d
			}
		}

		if delay == 0 {
			delay = 1 * time.Second
		}
		return newHttpServerDelayChaos(r , delay) , nil
	case Request_error_type:
		return newHttpRequestHacker(r) , nil
	}
	return nil , errors.New("unknown http chaos type")
}
