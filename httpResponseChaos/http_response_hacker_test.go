package httpResponseChaos_test

import (
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"testing"
	"delve_tool/httpResponseChaos"
	"time"
	"context"
)

func TestHttpResponseHacker_Invade(t *testing.T) {
	log.InitLog(log.DebugLvl)
	defer log.Flush()
	client := rpc2.NewClient("127.0.0.1:8899")
	bks , err := client.ListBreakpoints()
	if err != nil{
		panic(err)
	}
	for _ , bk := range bks{
		client.ClearBreakpoint(bk.ID)
	}
	client.SetReturnValuesLoadConfig(&api.LoadConfig{})
	hacker := httpResponseChaos.NewHttpResponseHacker(client , 500)
	ctx := context.TODO()
	workTime := time.Second * 60
	err = hacker.Invade(ctx , workTime)
	if err != nil{
		panic(err)
	}
}
