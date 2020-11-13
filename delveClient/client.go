package delveClient

import (
	"delve_tool/types"
	"errors"
	"fmt"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"net"
	"time"
)

type DelveClient struct{
	client      *rpc2.RPCClient
	//服务端监听的地址
	address     string
}

//等待服务起来，获取连接
func waitServerToUp(address string)(*net.Conn , error){
	//等待三秒
	done := time.After(3 * time.Second)
	for{
		conn , err := net.DialTimeout("tcp", address,  1 *time.Second)
		if err == nil{
			return &conn , nil
		}
		log.Errorf("[waitServerToUp]Failed to dial address , error - %s , retrying...", err.Error())
		select {
		case <- done:
			temp := fmt.Sprintf("[waitServerToUp]Failed to connect to address , error - %s , quit client.",  err.Error())
			log.Errorf(temp)
			return nil, fmt.Errorf(temp)
		default:
		}
	}
}

func (dc *DelveClient) initClient (address string) error {
	log.Infof("[DelveClient.InitClient]Initing client....")
	if address == ""{
		log.Errorf("[DelveClient.InitClient]can't get delve server's address, please make sure the server is up.")
		return errors.New("[DelveClient.InitClient]can't get delve server's address, please make sure the server is up")
	}

	dc.address = address

	conn , err := waitServerToUp(address)
	if err != nil{
		return err
	}
	dc.client = rpc2.NewClientFromConn(*conn)
	dc.client.SetReturnValuesLoadConfig(&api.LoadConfig{
	})
	return nil
}

//清理所有的断点
func (dc *DelveClient) clearAllBreakPoints() error{
	log.Infof("[DelveClient.ClearAllBreakPoints] clearing all break points....")
	breakpoints , err := dc.client.ListBreakpoints()
	if err != nil{
		log.Errorf("[DelveClient.ClearAllBreakPoints]Failed to list breakpoints , error - %s", err.Error())
		return err
	}
	for _ , breakpoint := range breakpoints{
		_ , err = dc.client.ClearBreakpoint(breakpoint.ID)
		if err != nil{
			log.Errorf("[DelveClient.ClearAllBreakPoints]Failed to clear breakpoint at File&Line : %s:%d , error - %s", breakpoint.File, breakpoint.Line, err.Error())
			return err
		}
	}
	return nil
}

//根据参数初始化client
func (dc *DelveClient) InitAndWork(errorType types.ErrorType, workTime time.Duration , address string , params ...interface{}) error{
	defer func (){
		log.Infof("[DelveClient.setHttpRequestError]client disconnecting")
		if _ , err := dc.client.Halt() ; err != nil{
			log.Errorf("[DelveClient.InitAndWork]Failed to halt , error - %s", err.Error())
		}
		fmt.Printf("Client Halting....\n")
		if err := dc.client.Disconnect(false) ; err != nil{
			log.Errorf("[DelveClient.InitAndWork]Failed to disconnect client , error - %s", err.Error())
		}
	}()
	err := dc.initClient(address)
	if err != nil{
		return err
	}
	switch errorType{
	case types.SqlError:
		//为go-sql-driver注入异常
		str , ok := params[0].(string)
		if !ok{
			log.Errorf("please use a string type error info!")
			return errors.New("error info is not string")
		}
		fmt.Printf("[DelveClient.InitAndWork] Ready to set golang sql error , error info : %s\n" , str)
		return dc.setGolangSqlError(workTime, str)
	case types.HttpRequestError:
		fmt.Printf("[DelveClient.InitAndWork] Ready to set http request error....\n")
		return dc.setHttpRequestError(workTime)
	case types.HttpStatusChaos:
		code , ok := params[0].(int)
		if !ok {
			log.Errorf("please use a int type status code!")
			return errors.New("status code is not int")
		}
		fmt.Printf("[DelveClient.InitAndWork] Ready to set http status chaos....\n")
		return dc.setHttpResponseHacker(workTime, code)
	default:
		//如果是未知类型，就会调用此方法让server提前退出，防止server一直监听导致协程不退出
		log.Errorf("[DelveClient.InitAndWork]Unknown errorType!")
		return errors.New("[DelveClient.InitAndWork]Unknown errorType")
	}
}





