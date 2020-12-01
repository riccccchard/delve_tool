package delveServer

import (
	"net"
	"time"

	"delve_tool/log"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpccommon"
	"fmt"
)

type DelveServer struct {
	server service.Server
	//接受停止信号
	disconnectCH chan struct{}
	//监听地址，默认为127.0.0.1:12345
	address string
	//需要attach的pid
	attachPid int
	//工作的最多时长，与客户端的时长有关
	duration time.Duration
}

func (ds *DelveServer) GetAddress() string {
	return ds.address
}
func (ds *DelveServer) SetAddress(address string) {
	ds.address = address
}
func (ds *DelveServer) GetAttachPid() int {
	return ds.attachPid
}

//根据信息初始化server，address为server监听端口，
//acceptMulti表示server是否支持多次client连接，为false表示如果一个client disconnect，那么server也会退出；
//目前由于Delve server不可多个client重入，所以acceptMulti为true会发生delve server退出不了的情况
//所以acceptMulti默认为false
func (ds *DelveServer) InitServer(attachPid int, address string, duration time.Duration) error {
	log.Infof("initing delve server with pid : %d , listen address : %s", attachPid, address)
	fmt.Printf("initing delve server with pid : %d , listen address : %s\n", attachPid, address)

	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Errorf("Failed to listen to address : %s , error : %s", address, err.Error())
		fmt.Printf("Failed to listen to address : %s , error : %s\n", address, err.Error())
		return err
	}
	ds.SetAddress(address)
	ds.attachPid = attachPid

	ds.disconnectCH = make(chan struct{})

	ds.duration = duration

	workingDir := "."
	config := &service.Config{
		Listener:           listener,
		ProcessArgs:        []string{},
		AcceptMulti:        false,
		APIVersion:         2,
		CheckLocalConnUser: false,
		DisconnectChan:     ds.disconnectCH,
		Debugger: debugger.Config{
			AttachPid:            attachPid,
			WorkingDir:           workingDir,
			Backend:              "default",
			CoreFile:             "",
			Foreground:           true,
			Packages:             nil,
			BuildFlags:           "",
			ExecuteKind:          debugger.ExecutingOther,
			DebugInfoDirectories: nil,
			CheckGoVersion:       true,
			TTY:                  "",
			Redirects:            [3]string{}, //可以重定向server的I/O信息
		},
	}
	ds.server = rpccommon.NewServer(config)
	return nil
}

//启动一个serer去attach 目标进程，同时监听停止信号
func (ds *DelveServer) StartServer() error {
	log.Infof("starting delve attach server.")
	fmt.Printf("starting delve attach server.\n")
	if err := ds.server.Run(); err != nil {
		log.Errorf("Failed to run server , error : %s", err.Error())
		return err
	}
	log.Infof("delve server attached to pid : %d , listen address : %s", ds.attachPid, ds.address)
	fmt.Printf("delve server attached to pid : %d , listen address : %s\n", ds.attachPid, ds.address)
	return nil
}

//监听停止信号
func (ds *DelveServer) WaitForStopServer() error {
	log.Infof("Waiting for Server stop.")
	fmt.Printf("Waiting for Server stop.\n")
	ticker := time.NewTicker(ds.duration)
	select {
	case <-ticker.C:
		log.Infof("server stoping by time ticker .")
		fmt.Printf("server stoping by time ticker .\n")
	case <-ds.disconnectCH:
		log.Infof("server stoping by client.")
		fmt.Printf("server stoping by client.\n")
	}

	//停止server
	err := ds.server.Stop()
	if err != nil {
		log.Errorf("failed to stop server : %s", err.Error())
		fmt.Printf("failed to stop server : %s\n", err.Error())
		return err
	}
	log.Infof("server stoped.")
	fmt.Printf("server stoped.\n")
	return nil
}
