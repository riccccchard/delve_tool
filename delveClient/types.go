package delveClient

/*
		错误类型定义
 */

type ErrorType  int

//错误类型表
const(
	//为go-sql-driver注入异常
	SqlError ErrorType = 0
)
