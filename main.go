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

	"golang.org/x/sync/errgroup"

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
	version = "0.4.2"
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
	fmt.Printf("[AttachTargetProcess] Initing Server ... \n")
	err := myDelveServer.InitServer(int(pid), address, duration+500*time.Millisecond) //比客户端多等0.5秒
	if err != nil {
		return err
	}
	fmt.Printf("[AttachTargetProcess] Staring Server ... \n")
	err = myDelveServer.StartServer()
	if err != nil {
		return err
	}
	fmt.Printf("[AttachTargetProcess] Waiting Server to stop... \n")
	err = myDelveServer.WaitForStopServer()
	if err != nil {
		return err
	}
	return nil
}

//注入故障
func SetErrorToTargetProcess(errorType int, duration time.Duration, address string) error {
	myDelveClient = &delveClient.DelveClient{}
	fmt.Printf("[SetErrorToTargetProcess] Client init and working... \n")
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
			fmt.Printf("version : %s\n", version)
			return
		}
	}
	log.InitLog(log.DebugLvl)
	log.Infof("[Main]Get args from command , pid : %d , address : %s , duration , %s , error type : %s", pid, address, duration, getErrorTypeString(delveClient.ErrorType(errorType)))

	if debug {
		setupSelveServerDebugLog()
	}

	if pid <= 0 {
		fmt.Printf("[Main]pid must be Positive number!\n")
		log.Errorf("[Main]pid must be Positive number!")
		flag.Usage()
		return
	}
	if duration <= 0 {
		log.Infof("[Main]duration is a negative integer , force it to 10 seconds.")
		duration = 10 * time.Second
	}
	fmt.Printf("[Main]Starting to attach process and set up client...\n")
	g := &errgroup.Group{}
	g.Go(func() error {
		return AttachTargetProcess(uint32(pid), address)
	})

	g.Go(func() error {
		return SetErrorToTargetProcess(errorType, duration, address)
	})
	if err := g.Wait(); err != nil {
		log.Errorf("[Main]Failed to attach or wait server to stop...")
		return
	}
	log.Infof("[Main]Process done successful , quiting...")
	fmt.Printf("[Main]Process done successful , quiting...\n")
	log.Flush()
}
