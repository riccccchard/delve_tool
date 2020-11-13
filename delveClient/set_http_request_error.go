package delveClient

import (
	"context"
	"delve_tool/httpRequestChaos"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"time"
)

/*
	修改net/http/server.go.(*conn).readRequest的返回值
*/

func (dc *DelveClient) setHttpRequestError ( workTime time.Duration) error{
	log.Infof("[DelveClient.setHttpRequestError]start set http request error....")

	hacker := httpRequestChaos.NewHttpRequestHacker(dc.client)

	ctx := context.TODO()

	err := hacker.Invade(ctx , workTime)
	if err != nil{
		return err
	}
	return nil
}