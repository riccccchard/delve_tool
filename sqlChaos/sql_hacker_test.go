package sqlChaos_test

import (
	"context"
	"delve_tool/log"
	"delve_tool/sqlChaos"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"testing"
	"time"
)


func TestSqlQueryHacker_Invade(t *testing.T) {
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
	hacker , _ := sqlChaos.NewSqlChaos(client , "query_error" , "you are hacked")
	ctx := context.TODO()
	workTime := time.Second * 30
	err = hacker.Invade(ctx , workTime)
	if err != nil{
		panic(err)
	}
}
