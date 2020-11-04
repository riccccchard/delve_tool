package delveClient

import (
	"context"
	"delve_tool/sqlChaos"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"time"
)

/*
		修改db.Query的error返回值
 */

//设置go-sql-driver库中对应function的错误返回值
//funcname : 需要hack的function名称
func (dc *DelveClient) setGolangSqlError ( workTime time.Duration) error{
	log.Infof("[DelveClient.SetMysqlQueryError]start set sql query error....")

	hacker := sqlChaos.NewSqlHacker(dc.client)

	ctx := context.TODO()

	err := hacker.Invade(ctx , workTime )
	if err != nil{
		return err
	}

	if err = dc.client.Disconnect(true) ; err != nil{
		log.Errorf("[DelveClient.setGolangSqlError]Failed to disconnect , error - %s", err.Error())
		return err
	}
	log.Infof("[DelveClient.SetSqlQueryError]client disconnect....")
	return nil
}