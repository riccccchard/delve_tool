package containerClient

import (
	"context"
	dockerclient "github.com/docker/docker/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"fmt"
)

func TestContainerClient_GetContainerID(t *testing.T) {
	config , err := clientcmd.BuildConfigFromFlags("", "/Users/xiaoshupeng/.kube/config")
	if err != nil{
		panic(err)
	}
	namespace := "default"
	podName := "httppidget-546c7f8854-6hnrb"
	containerName := "httppidget"
	k8sClient , err := client.New(config , client.Options{})
	if err != nil{
		panic(err)
	}
	list := &v1.PodList{}
	err = k8sClient.List(context.Background(), list , client.ListOption(&client.ListOptions{Namespace: namespace, Limit: 500}))
	if err != nil{
		panic(err)
	}
	for _ , pod := range list.Items{
		if pod.Name == podName{
			for _ , container := range pod.Status.ContainerStatuses{
				if containerName == container.Name{
					msg := fmt.Sprintf("Get container id : %s ", container.ContainerID)
					t.Log(msg)
					return
				}
			}
		}
	}
	t.Error("Can't find")
}

func TestContainerClient_GetPidFromContainerID(t *testing.T) {
	id := "0862ac01452435352233630ba19ec9415a50f3b66fb006562c5439c0f9de8017"

	dockerClient , err :=  dockerclient.NewClient(defaultDockerSocket, "", nil, nil)

	if err != nil{
		panic(err)
	}

	container , err := dockerClient.ContainerInspect(context.Background() , id)
	if err != nil{
		panic(err)
	}
	t.Log(container.State.Pid)
}