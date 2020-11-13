package httpRequestChaos_test

import (
	"delve_tool/httpRequestChaos"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"testing"
	"context"
	"time"
)

func TestHttpRequestHacker_Invade(t *testing.T) {
	log.InitLog(log.DebugLvl)
	client := rpc2.NewClient("127.0.0.1:8899")
	client.SetReturnValuesLoadConfig(&api.LoadConfig{})
	hacker := httpRequestChaos.NewHttpRequestHacker(client)

	ctx := context.TODO()
	workTime := time.Second * 60
	err := hacker.Invade(ctx , workTime)
	if err != nil{
		panic(err)
	}
}
