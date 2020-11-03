package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	dockerclient "github.com/docker/docker/client"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	dockerEndpoints = "unix:///var/run/docker.sock"
)

var inCluster = true

func GetPidFromProcess(process string) (int, error) {
	cmd := fmt.Sprintf("ps -ef | grep %s | grep -v 'grep' | grep -v 'process' | awk '{print $2}'", process)
	output, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return 0, err
	}
	pids := strings.Split(string(output), "\n")
	if len(pids) == 0 {
		return 0, fmt.Errorf("no pid found for %v", process)
	}
	pid, err := strconv.ParseInt(pids[0], 10, 64)
	return int(pid), err
}

func GetPidFromPod(ctx context.Context, podName string, namespace string) (int, error) {
	if namespace == "" {
		namespace = "default"
	}
	var (
		config *rest.Config
		err    error
	)
	if inCluster {
		config, err = rest.InClusterConfig()
		if err != nil {
			return 0, err
		}
	} else {
		home := homedir.HomeDir()
		kubeconfig := filepath.Join(home, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return 0, err
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return 0, err
	}
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, v1.GetOptions{})
	if err != nil {
		return 0, err
	}

	containerID := strings.TrimPrefix(pod.Status.ContainerStatuses[0].ContainerID, "docker://")
	dockerClient, err := dockerclient.NewClient(dockerEndpoints, "", nil, nil)
	if err != nil {
		return 0, err
	}
	containerState, err := dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return 0, err
	}
	return int(containerState.State.Pid), nil
}
