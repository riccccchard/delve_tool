package httpServerChaos_test

import (
	"delve_tool/httpServerChaos"
	"delve_tool/log"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"testing"
	"context"
	"time"
)

func TestHttpServerDelay_Invade(t *testing.T) {
	log.InitLog(log.DebugLvl)
	client := rpc2.NewClient("127.0.0.1:8899")
	client.SetReturnValuesLoadConfig(&api.LoadConfig{})
	hacker , _ := httpServerChaos.NewHttpServerHacker(client , "delay" , 500 * time.Millisecond)

	ctx := context.TODO()
	workTime := time.Second * 60
	err := hacker.Invade(ctx , workTime)
	if err != nil{
		panic(err)
	}
}
