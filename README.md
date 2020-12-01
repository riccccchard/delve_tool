## delve tool：修改golang程序的返回值

### 在k8s中使用的条件

将该工具编译成二进制文件，在k8s的特权pod中运行，需要sys_ptrace和sys_admin的权限，同时需要指明hostPID来获取node的进程。（可以在pod部署时的yaml文件声明）

例子如下



```yaml

      .......
      #添加进程和网络的namespace特权
      hostNetwork: true
      hostIPC: true
      hostPID: true
      ........
          capabilities:
            add:
                #添加pod的特权
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
      ,.........


```



###  二进制文件的使用和参数说明

使用如下命令编译为二进制文件

```go
CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o delve_tool
```

二进制文件的参数如下，也可以使用./delve_tool --help查看。

1. 通用参数

   通用参数包含了一些所有类型都有效的参数，其中pid为必填项

   1. Pid：需要attach 的目标进程pid
   2. Address：delve server监听的地址，默认为127.0.0.1:0，即随机端口
   3. Duration：混沌实验需要经历的时间，默认为30s
   4. Debug：是否打印delve debugger本身自带的日志

2. sql chaos参数

   sql chaos参数包含了有关数据库的异常类型，包括

   1. Type：包括"delay" ， "query_error" , "conn_pool"三种

   2. delay：delay表示对数据库操作注入延迟操作，包括增删改查，可以指明delay时间，使用示例：

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 sqlchaos --type=delay --delay=500ms
      ```

      表示对pid=10000的进程执行sqlchaos 的延迟操作，延迟时间为500ms

   3. Query_error：表示对数据库的查询操作注入故障，修改查询操作的返回值error，使用示例

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 sqlchaos --type=query_error --errorInfo="you are hacked"
      ```

      表示对pid=10000的进程执行sqlchaos的query_error类型，如果执行查询操作，如database/sql.(*Db).Query()，则会返回一个非空的error，并且其字符串信息为"you are hacked"

   4. Conn_pool：conn_pool表示对数据库连接池注入异常，实现方法就是启动number个协程去连接目标mysql数据库。

      使用示例

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 sqlchaos --type=conn_pool --number=100 --mysqlinfo="user:password@tcp(127.0.0.1:3306)/user"
      ```

      

3. http chaos参数

   （注意，这里的http chaos是使用在服务端的，而不是客户端）

   http chaos参数包含了有关http的异常类型，包括

   1. Type：包括"delay" , "request_error" 两种类型

   2. Delay：http server处理请求的延迟，示例

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 httpchaos --type=delay --delay=500ms
      ```

      表示将pid=10000的进程的http server处理请求的过程延迟500ms

   3. Request_error：表示服务端对所有到来的请求返回 400 bad request

      使用示例：

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 httpchaos --type=request_error 
      ```

4. gRPC chaos 参数

   （注意，这里的gRPC chaos注入在服务端而不是客户端）

   1. Type：包括"delay" "response_error"（response_error的实现还需要讨论，目前没有实现）

   2. Delay：gRPC处理请求的延迟，示例：

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 grpcchaos --type=request_error 
      ```

5. function chaos参数

   Function chaos参数可以在指定的函数或者代码行处添加延迟或者直接panic

   1. type： 包括"delay"和"panic"

   2. Delay：表示在特定的函数入口处或者代码行处注入延迟，

      functioneNames或者lines要用逗号隔开

      使用示例

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 functionchaos --type=delay --functionNames=database/sql.(*DB).Query,database/sql.(*DB).Exec 
      --lines=database/sql/sql.go:1548,database/sql/sql.go:1549
      ```

   3. Panic：表示让程序在特定的函数入口处或者代码行处抛出panic退出

      使用示例

      ```bash
      ./delve_tool --pid=10000 --duration=10s --debug=true  --address=127.0.0.1:30303 functionchaos --type=panic --functionNames=database/sql.(*DB).Query,database/sql.(*DB).Exec 
      --lines=database/sql/sql.go:1548,database/sql/sql.go:1549
      ```





