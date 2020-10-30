## delve tool：修改golang程序的返回值

### 使用

将该工具编译成二进制文件，在k8s的特权pod中运行，需要sys_ptrace和sys_admin的权限，同时需要指明hostPID来获取node的进程。（可以在pod部署时的yaml文件声明）

例子如下

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: delveserver
  namespace: default
  labels:
    app: delveserver
spec:
  selector:
    matchLabels:
      name: delveserver
  template:
    metadata:
      labels:
        name: delveserver
    spec:
      hostNetwork: true
      hostIPC: true
      hostPID: true
      containers:
      - name: delveserver
        image: delveserver
        imagePullPolicy: Never
        ports:
        - containerPort: 3333
        securityContext:
          privileged: true
          capabilities:
            add:
              - SYS_PTRACE
              - SYS_ADMIN
        volumeMounts:
          - name: socket-path
            mountPath: /var/run/docker.sock
            # mountPath: /run/containerd/containerd.sock，如果使用containerd的话
          - name: sys-path
            mountPath: /sys
        resources:
          limits:
            memory: "500Mi"
          requests:
            memory: "100Mi"
      - name: delveclient
        image: delveclient
        imagePullPolicy: Never
        ports:
        - containerPort: 8888
        resources:
          limits:
            memory: "500Mi"
          requests:
            memory: "100Mi"
      terminationGracePeriodSeconds: 30

      volumes:
        - name: socket-path
          hostPath:
            path: /var/run/docker.sock
        - name: sys-path
          hostPath:
            path: /sys

```



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

container：你想要attach的container名字，都是可以通过kubectl describe拿到的，默认为pod中的第一个container.

namespace：你想要attach的pod所在的namespace，默认为"default"

address：将要启动的delve server所监听的地址，默认为127.0.0.1:30303

duration：整个attach需要持续的时间，默认为30s

containerRuntime：k8s底层的container runtime interface的实现方法，目前支持"docker"和"containerd"两种，默认为docker

type：你想要注入的故障类型，0表示数据库查询异常

##  使用建议

在k8s的daemonSet中启动特权容器，并使用backgourndProcessManager启动一个新进程运行二进制文件并管理之，在启动的时候给他传入参数即可。

##  实例：修改db.Query的返回值

1. 首先，启动一个http服务，每次请求都会做一次sql查询操作。

```go
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var (
	myerr = errors.New("this is not an error")
)

type User struct {
	ID       int64  `json:"id"`
	UserName string `json:"username"`
	NickName string `json:"nickname"`
	PassWord string `json:"password"`
	Status   int    `json:"status"`
}

func main() {
	mysqlService := getMysqlService()
	dataSource := fmt.Sprintf("root:q755100802@tcp(%s)/user_db", mysqlService)
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		panic(err)
	}

	fmt.Println("why?")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//query := "SELECT id, uid, user_name, nick_name FROM user_base_info_tab_00000000 LIMIT 3"
		query := "Select * from user_db where username='test'"
		rows, err := db.Query(query)

		if err != nil {
			fmt.Fprintln(w, err.Error())
			return
		}
		bt, _ := json.Marshal(scan(rows))
		fmt.Fprintln(w, string(bt))
	})
	http.ListenAndServe(":9100", nil)
}

func scan(rows *sql.Rows) []User {
	if rows == nil {
		return nil
	}
	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.UserName, &user.NickName, &user.PassWord, &user.Status)
		if err != nil {
			fmt.Println("error: ", err.Error())
			continue
		}
		users = append(users, user)
	}
	return users
}

//获取mysql service的host和port
func getMysqlService() string {
	mysqlServiceHost := os.Getenv("MYSQL_SERVICE_SERVICE_HOST")
	mysqlServicePort := os.Getenv("MYSQL_SERVICE_SERVICE_PORT")
	return mysqlServiceHost + ":" + mysqlServicePort
}

func getMysqlService2() string {
	return "127.0.0.1:3306"
}

```

可以通过deployment和service将其部署在k8s中，对外暴露http服务端口即可。

代码和deployment文件放在example文件夹中。

这样，可以在本机上通过curl localhost:30307访问服务

![img](https://lh3.googleusercontent.com/-E4xYMrzGNs1-VZ_KskmJTeJGN4B-Y6Wb3mmQIpRb9y2R8oZgGCj1SyP1LUbk9wVz6sn97vv2AOtm1UHyDNfWkdZLAISUguVK-9Qq2dQvoK6DH7MYZ_EqFbg0RBBicsoaX6nlNI--IM)

2. 随后，在特权进程中安装delve_tool，如exec进入chaos-daemon，然后执行

   ```bash
   wget https://github.com/riccccchard/delve_tool/releases/download/delve_tool-0.2.0/delve_tool
   ```

   请确保有权限执行这个二进制文件：

   ```bash
   chmod 777 delve_tool
   ```

3. 使用这个delve_tool去attach目标container

   ```go
   ./delve_tool --pod="httpapp-68d9c99659-qclmt" --container="httpapp" --namespace="default" --address="127.0.0.1:3030" --duration=10s --containerRuntime="docker" --type=0
   ```

   （请根据readme.md中的参数介绍填写自己的参数）

4. 在之后的10s中的时间内，采用curl就会返回我们想要的结果：

   ![image-20201030153254145](/Users/xiaoshupeng/delve_tool/image/demo.png)





