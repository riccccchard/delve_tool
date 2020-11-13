package main

import (
	"context"
	"delve_tool/containerClient"
	"delve_tool/delveClient"
	"delve_tool/delveServer"
	"delve_tool/types"
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
	//自定义注入的error 信息
	errorInfo     string
	//有关http的status code信息
	httpStatusCode int
)


const (
	version = "1.0.0"
)

func init() {
	//随机端口
	flag.StringVar(&address, "address", "127.0.0.1:0", "address that delve server listen on")
	flag.DurationVar(&duration, "duration", 30*time.Second, "Duration of the experiment")
	flag.IntVar(&errorType, "type", 0, types.GetErrorUsage())
	flag.IntVar(&pid, "pid", 0, "target process pid")
	flag.BoolVar(&debug, "debug", false, "debug is used to print delve server and rpc-json flags")
	flag.StringVar(&errorInfo , "errorInfo" , "" , "errorInfo defines the error information that will be used to inject.")
	flag.IntVar(&httpStatusCode, "httpStatusCode" , 500 , "http status code defines the status code that will be used to inject to response.")
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
func SetErrorToTargetProcess(errorType types.ErrorType, duration time.Duration, address string , params ...interface{}) error {
	myDelveClient = &delveClient.DelveClient{}
	fmt.Printf("[SetErrorToTargetProcess] Client init and working... \n")
	return myDelveClient.InitAndWork(errorType, duration, address , params)
}

func getErrorTypeString(errorType types.ErrorType) string {
	switch errorType {
	case types.SqlError:
		return "sql-error"
	case types.HttpRequestError:
		return "http-request-error"
	case types.HttpStatusChaos:
		return "http-status-chaos"
	}
	return "unknow error type"
}

//打开delve server调试信息
func setupDelveServerDebugLog() {
	logflags.Setup(true, "debugger", "")
}

func checkoutArguementCorrect() bool {
	if pid <= 0 {
		fmt.Printf("[checkoutArguementCorrect]pid must be Positive number!\n")
		log.Errorf("[checkoutArguementCorrect]pid must be Positive number!")
		flag.Usage()
		return false
	}
	_errorType := types.ErrorType(errorType)
	if _ , ok := types.ChaosTypeMap[_errorType]; !ok{
		fmt.Printf("[checkoutArguementCorrect]Unknown error type!\n")
		log.Errorf("[checkoutArguementCorrect]Unknown error type!")
		flag.Usage()
		return false
	}
	if duration <= 0 {
		fmt.Printf("[checkoutArguementCorrect]duration is a negative integer , force it to 10 seconds.\n")
		log.Infof("[checkoutArguementCorrect]duration is a negative integer , force it to 10 seconds.")
		duration = 10 * time.Second
	}

	if _errorType == types.HttpStatusChaos && (httpStatusCode < 100  && httpStatusCode > 999){
		fmt.Printf("[checkoutArguementCorrect]error http status code setting! force it to 500\n")
		httpStatusCode = 500
	}
	return true
}

func main() {
	flag.Parse()
	help := false
	for i := range os.Args {
		if os.Args[i] == "version" || os.Args[i] == "v" || os.Args[i] == "-v" {
			fmt.Printf("version : %s\n", version)
			help=true
		}
		if os.Args[i] == "help" {
			flag.Usage()
			help=true
		}
	}
	if help{
		return
	}
	log.InitLog(log.DebugLvl)
	defer log.Flush()
	_errorType := types.ErrorType(errorType)
	log.Infof("[Main]Get args from command , pid : %d , address : %s , duration , %s , error type : %s", pid, address, duration, getErrorTypeString(_errorType))

	if debug {
		setupDelveServerDebugLog()
	}
	if !checkoutArguementCorrect(){
		return
	}
	fmt.Printf("[Main]Starting to attach process and set up client...\n")
	g := &errgroup.Group{}
	g.Go(func() error {
		return AttachTargetProcess(uint32(pid), address)
	})

	g.Go(func() error {
		switch _errorType {
		case types.SqlError:
			return SetErrorToTargetProcess(_errorType, duration, address , errorInfo)
		case types.HttpRequestError:
			return SetErrorToTargetProcess(_errorType, duration, address)
		case types.HttpStatusChaos:
			return SetErrorToTargetProcess(_errorType, duration , address , httpStatusCode)
		default:
			return nil
		}
	})
	if err := g.Wait(); err != nil {
		log.Errorf("[Main]Failed to attach or wait server to stop...")
		return
	}
	log.Infof("[Main]Process done successful , quiting...")
	fmt.Printf("[Main]Process done successful , quiting...\n")
}
