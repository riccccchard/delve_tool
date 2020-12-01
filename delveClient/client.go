package delveClient

import (
	"delve_tool/log"
	"errors"
	"fmt"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"net"
	"time"
)

//等待服务起来，获取连接
func waitServerToUp(address string)(*net.Conn , error){
	//等待三秒
	done := time.After(3 * time.Second)
	for{
		conn , err := net.DialTimeout("tcp", address,  1 *time.Second)
		if err == nil{
			return &conn , nil
		}
		log.Errorf("Failed to dial address , error - %s , retrying...", err.Error())
		fmt.Printf("Failed to dial address , error - %s , retrying...\n", err.Error())
		time.Sleep(100 * time.Millisecond)
		select {
		case <- done:
			temp := fmt.Sprintf("Failed to connect to address , error - %s , quit client.",  err.Error())
			log.Errorf(temp)
			return nil, fmt.Errorf(temp)
		default:
		}
	}
}

func InitClient(address string) ( *rpc2.RPCClient , error) {
	log.Infof("Initing client....")
	if address == ""{
		log.Errorf("can't get delve server's address, please make sure the server is up.")
		return nil, errors.New("can't get delve server's address, please make sure the server is up")
	}
	conn , err := waitServerToUp(address)
	if err != nil{
		return nil, err
	}
	var client *rpc2.RPCClient
	ch := make(chan bool)
	//做一个逻辑：如果超过2秒，说明server在启动之后就挂掉了，此时NewClientFromConn会卡住，需要退出
	go func(){
		go func(){
			timer := time.NewTicker(5 * time.Second)
			<- timer.C
			//执行到这里说明超时
			ch <- false
		}()
		client = rpc2.NewClientFromConn(*conn)
		client.SetReturnValuesLoadConfig(&api.LoadConfig{
		})
		//执行到这里说明成功
		ch <- true
	}()
	if ok := <- ch ; !ok{
		return nil , errors.New("Failed to new client from conn , is server already quit? ")
	}
	close(ch)
	return client,nil
}
