package main

import (
	"context"
	"delve_tool/containerClient"
	"delve_tool/delveClient"
	"delve_tool/delveServer"
	"flag"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"golang.org/x/sync/errgroup"
	"time"
	"fmt"
)
var(
	podName             string
	containerName       string
	namespace           string
	containerRuntime    string
	duration            time.Duration
	errorType           int
	myDelveServer       *delveServer.DelveServer
	address             string
	myDelveClient       *delveClient.DelveClient
)
const (
	errorTypeUsage =
`experiment's error type
0 : sql query error
`
)
func init(){
	flag.StringVar(&podName , "pod", "", "the name of target pod")
	flag.StringVar(&containerName , "container", "", "the name of target container")
	flag.StringVar(&namespace, "namespace", "default", "the namespace of target pod")
	flag.StringVar(&address , "address", "127.0.0.1:30303", "address that delve server listen on")
	flag.DurationVar(&duration, "duration", 30 * time.Second, "Duration of the experiment")
	flag.StringVar(&containerRuntime, "containerRuntime" , "docker" , "container runtime interface type ,now support docker and containerd")
	flag.IntVar(&errorType, "type", 0, errorTypeUsage)
}
//获取目标容器Pid
func GetTargetPid(podName string , namespace string , containerName string, containerRuntime string)( uint32, error){
	err := containerClient.ContainerRuntimeClient.CreateContainerClient(containerRuntime)
	if err != nil{
		return 0 , err
	}
	containerID , err := containerClient.ContainerRuntimeClient.GetContainerID(context.Background(), namespace , podName , containerName)
	if err != nil{
		return 0 , err
	}
	return containerClient.ContainerRuntimeClient.GetPidFromContainerID(containerID)

}

//启动delve server attach目标进程，wairForStopServer是阻塞的，需要起协程
func AttachTargetProcess(pid uint32, address string)error{
	myDelveServer = &delveServer.DelveServer{}

	err := myDelveServer.InitServer(int(pid), address , duration + 3*time.Second)//比客户端多等3秒
	if err != nil{
		return err
	}
	err = myDelveServer.StartServer()
	if err != nil{
		return err
	}
	err = myDelveServer.WaitForStopServer()
	if err != nil{
		return err
	}
	return nil
}

//注入故障
func SetErrorToTargetProcess(errorType int , duration time.Duration, address string)error{
	myDelveClient = &delveClient.DelveClient{}
	return myDelveClient.InitAndWork(delveClient.ErrorType(errorType) , duration , address)
}
func getErrorTypeString(errorType int)string{
	if errorType == 0 {
		return "sql query error"
	}
	return ""
}
func main(){
	flag.Parse()
	log.InitLog(log.InfoLvl)
	if podName == "" || containerName == "" {
		flag.Usage()
		return
	}
	fmt.Println(duration)
	log.Infof("[Main]Get args : namespace - %s , pod - %s , container - %s , duration - %v, errorType - %s , address - %s , containerRuntime - %s", namespace, podName, containerName , duration , getErrorTypeString(errorType), address , containerRuntime)

	g := &errgroup.Group{}

	pid , err := GetTargetPid(podName , namespace , containerName , containerRuntime)
	if err != nil{
		panic(err)
	}

	g.Go( func () error{
		return 	AttachTargetProcess(pid , address)
	})

	g.Go( func() error{
		return  SetErrorToTargetProcess(errorType , duration , address)
	})

	if err = g.Wait(); err != nil{
		panic(err)
	}
}
