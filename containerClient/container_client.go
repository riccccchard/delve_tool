package containerClient

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	dockerclient "github.com/docker/docker/client"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	"github.com/containerd/containerd"
	"context"
	"net/http"
	"errors"
)

var(
	ContainerRuntimeClient *ContainerClient
)

const(
	containerRuntimeDocker     = "docker"
	containerRuntimeContainerd = "containerd"

	defaultDockerSocket  = "unix:///var/run/docker.sock"
	dockerProtocolPrefix = "docker://"

	defaultContainerdSocket  = "/run/containerd/containerd.sock"
	containerdProtocolPrefix = "containerd://"
	containerdDefaultNS      = "k8s.io"

	defaultProcPrefix = "/proc"
)

//负责与kube-apiserver和container runtime interface交互
type ContainerClient struct{
	k8sClient           client.Client
	dockerClient        DockerClientInterface
	containerdClient    ContainerdClientInterface
	containerRuntime    string
}

type DockerClientInterface interface {
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}

type ContainerdClientInterface interface {
	LoadContainer(ctx context.Context, containerId string)(containerd.Container, error)
}

//将 "docker://"前缀的containerID 去除前缀，取出id
func (cc *ContainerClient) formatDockerContainerID(containerID string) (string , error){
	if len(containerID) < len(dockerProtocolPrefix) {
		log.Errorf("[ContainerClient.formatDockerContainerID]container id %s is not a docker container id", containerID)
		return "", fmt.Errorf("container id %s is not a docker container id", containerID)
	}
	if containerID[0:len(dockerProtocolPrefix)] != dockerProtocolPrefix {
		log.Errorf("[ContainerClient.formatDockerContainerID]expected %s but got %s", dockerProtocolPrefix, containerID[0:len(dockerProtocolPrefix)])
		return "", fmt.Errorf("expected %s but got %s", dockerProtocolPrefix, containerID[0:len(dockerProtocolPrefix)])
	}
	return containerID[len(dockerProtocolPrefix):], nil
}
//使用docker client获取pid
func (cc *ContainerClient) getPidFromDockerClient(ctx context.Context, containerID string) (uint32, error){
	log.Infof("[ContainerClient.getPidFromContainerdClient]get pid from docker client..., containerId : %s", containerID)
	id , err := cc.formatDockerContainerID(containerID)
	if err != nil{
		return 0, err
	}
	container , err := cc.dockerClient.ContainerInspect(ctx , id)
	if err != nil{
		log.Errorf("[ContainerClient.GetPidFromDockerContainerID]Failed to use docker client to get container, error - %s", err.Error())
		return 0 , err
	}
	return uint32(container.State.Pid), nil
}

//将 "containerd://"前缀的containerid 去除前缀
func (cc *ContainerClient) formatContainerdContainerID(containerID string) (string , error){
	if len(containerID) < len(containerdProtocolPrefix) {
		log.Errorf("[ContainerClient.formatContainerdContainerID]container id %s is not a containerd container id", containerID)
		return "", fmt.Errorf("container id %s is not a containerd container id", containerID)
	}
	if containerID[0:len(containerdProtocolPrefix)] != containerdProtocolPrefix {
		log.Errorf("[ContainerClient.formatContainerdContainerID]expected %s but got %s",containerdProtocolPrefix, containerID[0:len(containerdProtocolPrefix)])
		return "", fmt.Errorf("expected %s but got %s", containerdProtocolPrefix, containerID[0:len(containerdProtocolPrefix)])
	}
	return containerID[len(containerdProtocolPrefix):], nil
}

//使用containerd client获取pid
func (cc *ContainerClient) getPidFromContainerdClient(ctx context.Context , containerID string) (uint32 , error){
	log.Infof("[ContainerClient.getPidFromContainerdClient]get pid from container client..., containerId : %s", containerID)
	id , err := cc.formatContainerdContainerID(containerID)
	if err != nil{
		return 0 , err
	}
	container , err := cc.containerdClient.LoadContainer(ctx , id)
	if err != nil{
		log.Errorf("[ContainerClient.getPidFromContainerdClient]Failed to load container from containerd client , error - %s", err.Error())
		return 0 , err
	}
	task, err := container.Task(ctx, nil)
	if err != nil {
		log.Errorf("[ContainerClient.getPidFromContainerdClient]Failed to do task from container , error - %s" , err.Error())
		return 0, err
	}
	return task.Pid(), nil
}

func newDockerClient(host string, version string, client *http.Client, httpHeaders map[string]string) (DockerClientInterface, error) {
	return dockerclient.NewClient(host, version, client, httpHeaders)
}

// newContainerdClient returns a containerd.New with mock points
func newContainerdClient(address string, opts ...containerd.ClientOpt) (ContainerdClientInterface, error) {
	return containerd.New(address, opts...)
}

//根据containerRuntime指明使用哪个
func (cc *ContainerClient) CreateContainerClient(containerRuntime string)error{
	cc.containerRuntime = containerRuntime
	var err error
	switch containerRuntime{
	case containerRuntimeContainerd:
		log.Infof("[ContainerClient.CreateContainerClient]Create Containerd Client.")
		cc.containerdClient , err = newContainerdClient(defaultContainerdSocket, containerd.WithDefaultNamespace(containerdDefaultNS))
		if err != nil{
			log.Errorf("[ContainerClient.CreateContainerClient]Failed to new containerd client , error - %s", err.Error())
			return err
		}
	case containerRuntimeDocker:
		log.Infof("[ContainerClient.CreateContainerClient]create docker client.")
		cc.dockerClient , err = newDockerClient(defaultDockerSocket , "", nil, nil)
		if err != nil{
			log.Errorf("[ContainerClient.CreateContainerClient]Failed to new docker client , error - %s", err.Error())
			return err
		}
	default:
		return errors.New("[ContainerClient.CreateContainerClient]Unknow container runtime type")
	}

	cc.k8sClient , err = newK8sClient()
	if err != nil{
		return err
	}
	return nil
}

func (cc *ContainerClient) GetPidFromContainerID( containerID string) (uint32 , error){
	switch cc.containerRuntime{
	case containerRuntimeContainerd:
		return cc.getPidFromContainerdClient(context.Background() , containerID)
	case containerRuntimeDocker:
		return cc.getPidFromDockerClient(context.Background() , containerID)
	default:
		return 0 , errors.New("[ContainerClient.GetPidFromContainerID]Unkown container runtime type")
	}
}

