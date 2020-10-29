## delve tool：修改golang程序的返回值

### 使用

将该工具编译成二进制文件，在k8s的特权pod中运行，需要sys_ptrace和sys_admin的权限，同时需要指明hostPID来获取node的进程。（可以在pod部署时的yaml文件声明）

使用如下命令编译成二进制文件

```go
CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o delve_tool
```

使用时，使用如下命令运行二进制文件

```go
./delve_tool --pod="your pod name" --container="your container name" --namespace="your namespace" --address="127.0.0.1:30303" --duration=30s --containerRuntime="docker" --type=0
```

参数意义如下

pod：你想要attach的pod名字

container：你想要attach的container名字，都是可以通过kubectl describe拿到的

namespace：你想要attach的pod所在的namespace，默认为"default"

address：将要启动的delve server所监听的地址，默认为127.0.0.1:30303

duration：整个attach需要持续的时间，默认为30s

containerRuntime：k8s底层的container runtime interface的实现方法，目前支持"docker"和"containerd"两种，默认为docker

type：你想要注入的故障类型，0表示数据库查询异常

##  使用建议

在k8s的daemonSet中启动特权容器，并使用backgourndProcessManager启动一个新进程运行二进制文件并管理之，在启动的时候给他传入参数即可。

