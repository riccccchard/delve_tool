package types

/*
		错误类型定义
 */

type ErrorType  int

//错误类型表
const(
	//为go-sql-driver注入异常
	SqlError ErrorType = 0
	//为http request 注入异常
	HttpRequestError ErrorType = 1
	//为http response 设置状态码status code
	HttpStatusChaos ErrorType = 2
	//未知类型
	UnknownType ErrorType = -998244353

)
var (
	ChaosTypeMap = map[ErrorType]string {
		SqlError: sqlErrorDescribe,
		HttpRequestError: httpRequestErrorDescribe,
		HttpStatusChaos: httpStatusChaosDescribe,
    }
)
//错误类型介绍
const (
	sqlErrorDescribe = `
		sql error describe :  sql error will inject error to all of the database/sql operation,
							  such as database/sql.(*DB).Query, database/sql.(*DB).Exec , database/sql.(*Stmt).Exec and so on.
                              if errorinfo is not set , we will inject database/sql/driver.ErrBadConn to these functions.
							  else the error information will equal to  errorinfo.
	`
	httpRequestErrorDescribe = `
		http request error describe : http request error will inject error to the http server , so that the server will 
									  get error when reading request from client , now it can only return "http: 400 Bad Request"
	`
	httpStatusChaosDescribe = `
		http status chaos describe : http status chaos will change status code of response to something you want us to set.
								     if you don't set status code or the status code is not in range [100,999],
									 we will set the status code to 500.
	`
)


func GetErrorUsage() string{
	return sqlErrorDescribe + httpRequestErrorDescribe + httpStatusChaosDescribe
}
