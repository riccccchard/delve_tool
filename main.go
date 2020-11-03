package main

import (
	"context"
	"delve_tool/containerClient"
	"delve_tool/delveClient"
	"delve_tool/delveServer"
	"flag"
	"fmt"
	"os"
	"time"

	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"golang.org/x/sync/errgroup"
)

var (
	duration      time.Duration
	errorType     int
	myDelveServer *delveServer.DelveServer
	address       string
	myDelveClient *delveClient.DelveClient
	pid           int
)

const (
	errorTypeUsage = `experiment's error type
0 : sql query error
`
)

func init() {
	flag.StringVar(&address, "address", "127.0.0.1:30303", "address that delve server listen on")
	flag.DurationVar(&duration, "duration", 30*time.Second, "Duration of the experiment")
	flag.IntVar(&errorType, "type", 0, errorTypeUsage)
	flag.IntVar(&pid, "pid", 0, "target process pid")
}

//获取目标容器Pid
func GetTargetPid(podName string, namespace string, containerName string, containerRuntime string) (uint32, error) {
	err := containerClient.ContainerRuntimeClient.CreateContainerClient(containerRuntime)
	if err != nil {
		return 0, err
	}
	containerID, err := containerClient.ContainerRuntimeClient.GetContainerID(context.Background(), namespace, podName, containerName)
	if err != nil {
		return 0, err
	}
	return containerClient.ContainerRuntimeClient.GetPidFromContainerID(containerID)

}

//启动delve server attach目标进程，wairForStopServer是阻塞的，需要起协程
func AttachTargetProcess(pid uint32, address string) error {
	myDelveServer = &delveServer.DelveServer{}

	err := myDelveServer.InitServer(int(pid), address, duration+500*time.Millisecond) //比客户端多等0.5秒
	if err != nil {
		return err
	}
	err = myDelveServer.StartServer()
	if err != nil {
		return err
	}
	err = myDelveServer.WaitForStopServer()
	if err != nil {
		return err
	}
	return nil
}

//注入故障
func SetErrorToTargetProcess(errorType int, duration time.Duration, address string) error {
	myDelveClient = &delveClient.DelveClient{}
	return myDelveClient.InitAndWork(delveClient.ErrorType(errorType), duration, address)
}
func getErrorTypeString(errorType int) string {
	if errorType == 0 {
		return "sql query error"
	}
	return ""
}
func main() {
	flag.Parse()
	log.InitLog(log.DebugLvl)
	var err error
	log.Infof("[Main]Get args from command , pid : %d , address : %s , duration , %s , errorType : %s", pid, address, duration, getErrorTypeString(errorType))

	g := &errgroup.Group{}
	g.Go(func() error {
		return AttachTargetProcess(uint32(pid), address)
	})

	g.Go(func() error {
		return SetErrorToTargetProcess(errorType, duration, address)
	})
	//起一个协程计时，如果超过duration三秒直接停掉进程，防止因为其他原因阻塞在server.stop
	go func() {
		ticker := time.NewTicker(duration + 3*time.Second)
		select {
		case <-ticker.C:
			fmt.Printf("[Main]Process stoped by ticker , quiting...")
			log.Infof("[Main]Process stoped by ticker , quiting...")
			os.Exit(0)
		}
	}()
	if err = g.Wait(); err != nil {
		fmt.Printf("error : %s\n", err.Error())
		return
	}
	log.Infof("[Main]Process done...")
	fmt.Printf("Process done.")
}
