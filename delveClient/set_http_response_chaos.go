package delveClient

import (
	"context"
	"delve_tool/httpResponseChaos"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"time"
)

/*
	修改net/http/server.(*conn).server中的response的status code
	通过在server.(*response).finishRequest函数执行过程中，设置 response.status = your status code来实现
*/

func (dc *DelveClient) setHttpResponseHacker ( workTime time.Duration , statusCode int) error{
	log.Infof("[DelveClient.setHttpResponseHacker]start set http response error....")

	hacker := httpResponseChaos.NewHttpResponseHacker(dc.client , statusCode)

	ctx := context.TODO()

	err := hacker.Invade(ctx , workTime)
	if err != nil{
		return err
	}

	return nil
}