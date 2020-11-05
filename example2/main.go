package main

import (
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"xorm.io/xorm"
	"fmt"
	"os"
)

type User struct {
	ID       int64  `xorm:"id"`
	UserName string `xorm:"username"`
	NickName string `xorm:"nickname"`
	PassWord string `xorm:"password"`
	Status   int    `xorm:"status"`
}

func main(){
	serverInfo := fmt.Sprintf("root:root@tcp(%s)/user" , getMysqlService())

	engine, err := xorm.NewEngine("mysql" , serverInfo)

	if err != nil{
		panic(err)
	}

	err = engine.Sync2(new(User))
	if err != nil{
		panic(err)
	}

	fmt.Printf("starting http server....")
	count := 2
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user := &User{}
		ok, _ := engine.Where("username=?" , "test" ).Get(user)

		if ! ok {
			fmt.Fprintf(w , "Can't find any user username = test , error - %s\n", err.Error())
			return
		}

		fmt.Fprintf(w , "get user : %+v\n", user)
		user.Status = count
		_ , err := engine.Update(user)
		if err != nil{
			fmt.Fprintf(w , "Failed to update user status. , error - %s\n" , err.Error())
			return
		}
		count ++
	})
	http.ListenAndServe("0.0.0.0:9101" , nil)
}

//获取mysql service的host和port
func getMysqlService() string {
	mysqlServiceHost := os.Getenv("MYSQL_SERVICE_SERVICE_HOST")
	mysqlServicePort := os.Getenv("MYSQL_SERVICE_SERVICE_PORT")
	return mysqlServiceHost + ":" + mysqlServicePort
}