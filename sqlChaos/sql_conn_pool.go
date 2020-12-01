package sqlChaos

import (
	"context"
	"database/sql"
	"delve_tool/log"
	"sync"
	"time"
)

/*
		启动新协程，占用目标连接池连接
 */

type sqlConnPoolHacker struct{
	//需要启动的协程个数
	count int
	//mysql 连接信息 , 如  root:root@tcp(127.0.0.1:3306)/user
	serviceInfo string
}

func (h *sqlConnPoolHacker) Invade (ctx context.Context , timeout time.Duration) error{

	sctx , cancel := context.WithTimeout(ctx , timeout)

	db , err := sql.Open("mysql", h.serviceInfo)
	if err != nil{
		log.Error("Can't open serivce %s , error - %s", h.serviceInfo , err.Error())
		return err
	}
	defer db.Close()
	if err = db.Ping(); err != nil{
		log.Errorf("Can't ping service %s , error - %s", h.serviceInfo , err.Error())
		return err
	}

	if err := h.invade(sctx , timeout , db) ; err != nil{
		return err
	}
	cancel()
	return nil
}

func (h *sqlConnPoolHacker) invade (ctx context.Context, period time.Duration , db *sql.DB) error{
	log.Infof("start invading for %v", period)

	wg := new(sync.WaitGroup)

	wg.Add(h.count)

	for i := 0 ; i < h.count ; i ++ {
		go h.startConn(ctx , db , wg)
	}
	wg.Wait()
	db.Close()
	return nil
}

func (h *sqlConnPoolHacker) startConn(ctx context.Context , db *sql.DB , wg *sync.WaitGroup){
	defer wg.Done()
	queryStr := "SELETE * FROM sys LIMIT 10"
retry:
	rows , err := db.Query(queryStr)

	if err != nil{
		log.Errorf("db query error - %s", err.Error())
		goto retry
	}
	for rows.Next(){
		<- ctx.Done()
	}
}

func newConnPoolHacker(count int , serviceInfo string) *sqlConnPoolHacker{
	log.Infof("New conn pool chaos")
	hacker := &sqlConnPoolHacker{
		count: count,
		serviceInfo: serviceInfo,
	}
	return hacker
}
