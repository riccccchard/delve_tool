package grpcServerChaos

import (
	"delve_tool/types"
	"errors"
	"github.com/go-delve/delve/service/rpc2"
	"time"
)

const (
	Delay_type         = "delay"
	Request_error_type = "request_error"
)

func NewgRPCChaos(r *rpc2.RPCClient , chaosType string , param ...interface{})( types.ChaosInterface , error){
	switch chaosType {
	case Delay_type:
		var delay time.Duration
		if len(param) != 0{
			if d ,ok := param[0].(time.Duration) ; ok {
				delay = d
			}
		}
		if delay == 0 {
			delay = 1 * time.Second
		}
		return newGrpcServerDelayer(r , delay) , nil
	default:
		return nil , errors.New("unknown grpc chaos type")
	}
}