package containerClient

import (
	"context"
	"git.garena.com/shopee/loan-service/airpay_backend/public/common/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"errors"
)


//新建与kube-api交互的k8s client
func newK8sClient() (client.Client, error){
	config , err := rest.InClusterConfig()
	if err != nil{
		log.Errorf("[newK8sClient]Failed to read config in cluster , error - %s", err.Error())
		return nil , err
	}
	k8sClient , err := client.New(config , client.Options{})
	if err != nil{
		log.Errorf("[newk8sClient]Failed to new k8s client , error - %s", err.Error())
		return nil , err
	}
	return k8sClient, nil
}

func (cc *ContainerClient) GetContainerID (ctx context.Context , namespace string, podName string , containerName string) (string , error) {
	list := &v1.PodList{}
	err := cc.k8sClient.List(ctx, list , client.ListOption(&client.ListOptions{Namespace: namespace, Limit: 500}))
	if err != nil{
		return "" , err
	}
	for _ , pod := range list.Items{
		if pod.Name == podName{
			for _ , container := range pod.Status.ContainerStatuses{
				if containerName == "" && container.Name != "pause" {
					//如果没指明containerName，就返回第一个不是pause容器的containerID
					return container.ContainerID, nil
				}
				if containerName == container.Name{
					return container.ContainerID , nil
				}
			}
		}
	}
	return "" , errors.New("[ContainerClient.GetContainerID]Can't find target pod and container")
}


