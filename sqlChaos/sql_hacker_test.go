package sqlChaos_test

import (
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"testing"
	"time"
	"context"
	"delve_tool/sqlChaos"
)


func TestHttpRequestHacker_Invade(t *testing.T) {
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
	hacker := sqlChaos.NewSqlHacker(client , "you are hacked")
	ctx := context.TODO()
	workTime := time.Second * 60
	err = hacker.Invade(ctx , workTime)
	if err != nil{
		panic(err)
	}
}
