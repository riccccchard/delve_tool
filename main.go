package main

import (
	"context"
	"delve_tool/containerClient"
	"delve_tool/delveClient"
	"delve_tool/delveServer"
	"flag"
	"fmt"
	"golang.org/x/sync/errgroup"
	"os"
	"time"

	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/go-delve/delve/pkg/logflags"
)

var (
	duration      time.Duration
	errorType     int
	myDelveServer *delveServer.DelveServer
	address       string
	myDelveClient *delveClient.DelveClient
	pid           int
	//是否打印delve server 和rpc的调试信息
	debug bool
)

const (
	errorTypeUsage = `experiment's error type
0 : sql query error
`
	version = "0.4.1"
)

func init() {
	flag.StringVar(&address, "address", "127.0.0.1:30303", "address that delve server listen on")
	flag.DurationVar(&duration, "duration", 30*time.Second, "Duration of the experiment")
	flag.IntVar(&errorType, "type", 0, errorTypeUsage)
	flag.IntVar(&pid, "pid", 0, "target process pid")
	flag.BoolVar(&debug, "debug", false, "debug is used to pring delve server and rpc-json flags")
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

//启动delve server attach目标进程，waitForStopServer是阻塞的，需要起协程
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

func getErrorTypeString(errorType delveClient.ErrorType) string {
	switch errorType {
	case delveClient.SqlError:
		return "sql-error"
	}
	return "unknow error type"
}

//打开delve server调试信息
func setupSelveServerDebugLog() {
	logflags.Setup(true, "debugger", "")
}

func main() {
	flag.Parse()
	for i := range os.Args {
		if os.Args[i] == "version" || os.Args[i] == "v" || os.Args[i] == "-v" {
			fmt.Printf("version : %s", version)
			break
		}
	}
	log.InitLog(log.DebugLvl)
	log.Infof("[Main]Get args from command , pid : %d , address : %s , duration , %s , error type : %s", pid, address, duration, getErrorTypeString(delveClient.ErrorType(errorType)))

	if debug {
		setupSelveServerDebugLog()
	}

	if pid <= 0 {
		fmt.Printf("pid must be Positive number!")
		log.Errorf("pid must be Positive number!")
		flag.Usage()
		return
	}
	if duration <= 0 {
		log.Infof("duration is a negative integer , force it to 10 seconds.")
		duration = 10 * time.Second
	}

	g := &errgroup.Group{}
	g.Go( func () error {
		return AttachTargetProcess(uint32(pid), address)
	})

	g.Go( func () error {
		return SetErrorToTargetProcess(errorType , duration , address)
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
	if err := g.Wait() ; err != nil{
		log.Errorf("[Main]Failed to attach or wait server to stop...")
		return
	}
	log.Infof("[Main]Process done successful , quiting...")
	fmt.Printf("[Main]Process done successful , quiting...")
}
