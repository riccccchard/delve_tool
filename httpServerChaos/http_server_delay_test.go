package httpServerChaos_test

import (
	"delve_tool/httpServerChaos"
	"delve_tool/log"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"time"
	"testing"
	"context"
)


func TestHttpDelayer_Invade(t *testing.T) {
	log.InitLog(log.DebugLvl)
	defer log.Flush()
	client := rpc2.NewClient("127.0.0.1:8899")
	client.SetReturnValuesLoadConfig(&api.LoadConfig{})
	bks , err := client.ListBreakpoints()
	if err != nil{
		panic(err)
	}
	for _ , bk := range bks{
		client.ClearBreakpoint(bk.ID)
	}
	hacker, _ := httpServerChaos.NewHttpServerHacker(client , "delay" , 500 * time.Millisecond)
	ctx := context.TODO()
	workTime := time.Second * 30
	err = hacker.Invade(ctx , workTime)
	if err != nil{
		panic(err)
	}
}
