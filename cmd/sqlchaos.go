/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"delve_tool/delveClient"
	"delve_tool/log"
	"delve_tool/sqlChaos"
	"delve_tool/types"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"time"
)

// sqlchaosCmd represents the sqlchaos command
var sqlchaosCmd = &cobra.Command{
	Use:   "sqlchaos",
	Short: "sql chaos for golang go-sql-driver",
	Long: `sql chaos for golang go-sql-driver , including sql option delay add conn pool full.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.InitLog(log.DebugLvl)
		defer log.Flush()
		if !checkoutArguementCorrect(){
			return
		}
		if debug{
			setupDelveServerDebugLog()
		}
		printMainArgs()

		fmt.Printf("Starting to attach process and set up client...\n")
		log.Infof("Starting to attach process and set up client...")

		g := &errgroup.Group{}
		g.Go(func() error {
			return AttachTargetProcess(uint32(pid) , address)
		})

		g.Go(func() error {
			return initAndRunSqlChaos()
		})

		if err := g.Wait() ; err != nil{
			log.Errorf("Failed to attach or wait server to stop... , error - %s", err.Error())
			fmt.Printf("Failed to attach or wait server to stop... , error - %s\n", err.Error())
			return
		}
		log.Infof("[Main]Process done successful , quiting...")
		fmt.Printf("[Main]Process done successful , quiting...\n")
	},
}

/*
		根据入参选择需要执行的类型，如果同时指明，则报错
 */

var(
	sqlchaosType        string
	//当为query_error时，指明query的返回错误信息
	sqlchaosErrorInfo   string
	//当为conn_pool时，指明需要另起多少个连接
	sqlchaosConnNumber  int
	//当为conn_pool时，指明需要连接的mysql service信息
	mysqlInfo  string
	//当为delay type时，指明需要延迟的时间
	sqlchaosDelay       time.Duration
)
func init() {
	rootCmd.AddCommand(sqlchaosCmd)
	//sql chaos type 默认为delay

	sqlchaosCmd.Flags().StringVar(&sqlchaosType , "type", "delay" ," the type of sql chaos, including three types , \"delay\" , \"conn_pool\" , \"query_error\"")
	sqlchaosCmd.Flags().StringVar(&sqlchaosErrorInfo , "errorInfo", "" , "the error information you want to modify when call a database query")
	sqlchaosCmd.Flags().IntVar(&sqlchaosConnNumber , "number" , 100, "the number of connection you want to inject to database")
	sqlchaosCmd.Flags().StringVar(&mysqlInfo , "mysqlinfo" , "127.0.0.1:3306" , "the information of mysql which you want to connect , like user:password@tcp(127.0.0.1:3306)/user")
	sqlchaosCmd.Flags().DurationVar(&sqlchaosDelay , "delay" , 500 * time.Millisecond , "delay time to sql chaos")
}

func initAndRunSqlChaos() error{
	client , err := delveClient.InitClient(address)
	if err != nil{
		log.Errorf("Failed to init client , error - %s" , err.Error())
		fmt.Printf("Failed to init client , error = %s\n" , err.Error())
		return err
	}
	var hacker types.ChaosInterface
	switch sqlchaosType{
	case sqlChaos.Delay_type:
		hacker , err = sqlChaos.NewSqlChaos(client , sqlchaosType , sqlchaosDelay)

	case sqlChaos.Query_error_type:
		hacker , err = sqlChaos.NewSqlChaos(client , sqlchaosType , sqlchaosErrorInfo)

	case sqlChaos.Conn_pool_type:
		hacker , err = sqlChaos.NewSqlChaos(client , sqlchaosType , sqlchaosConnNumber , mysqlInfo)
	default:
		log.Errorf("unknown sql chaos type")
		fmt.Printf("unknown sql chaos type\n")
		return errors.New("unknown sql chaos type")
	}
	if err != nil{
		log.Errorf("Failed to new sql chaos , error - %s", err.Error())
		fmt.Printf("Failed to new sql chaos , error - %s\n", err.Error())
		return err
	}
	if err = hacker.Invade(context.Background() , duration) ; err != nil{
		log.Errorf("Failed to invade chaos , error - %s", err.Error())
		fmt.Printf("Failed to invade chaos , error - %s\n", err.Error())
		return err
	}

	log.Infof("invade sql chaos success")
	fmt.Printf("invade sql chaos success\n")
	return nil
}

