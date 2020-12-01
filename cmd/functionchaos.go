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
	"delve_tool/delveClient"
	"delve_tool/functionChaos"
	"context"
	"delve_tool/log"
	"delve_tool/types"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"strings"
	"time"
)

// functionchaosCmd represents the functionchaos command
var functionchaosCmd = &cobra.Command{
	Use:   "functionchaos",
	Short: "function chaos is used to inject chaos to function call ",
	Long: `function chaos is used to inject chaos to function call , including "delay" and "panic"`,
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
			return AttachTargetProcess(uint32(pid), address)
		})

		g.Go(func() error {
			return initAndRunFunctionChaos()
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

var(
	functionChaosType   string

	functionDelay       time.Duration

	//functionNames表示要inject chaos的function名字对象，需要输入全名，如database/sql.(*DB).Query ; 多个function names之间用逗号隔开
	functionNames       string

	//代码行表示要inject chaos的代码行地址，需要输入全名，如database/sql/sql.go:1547 ; 多个function names之间用逗号隔开
	lines               string
)
func init() {
	rootCmd.AddCommand(functionchaosCmd)

	functionchaosCmd.Flags().StringVar(&functionChaosType , "type", "delay", "type of function chaos")

	functionchaosCmd.Flags().DurationVar(&functionDelay , "delay", 500 * time.Millisecond , "delay time of delay type")

	functionchaosCmd.Flags().StringVar(&functionNames , "functionNames" , "" , "function target to inject chaos")

	functionchaosCmd.Flags().StringVar(&lines , "lines" , "" , "lines target to inject chaos")

}

func getFunctionNames(functionNames string) []string{
	return strings.Split(functionNames, ",")
}

func initAndRunFunctionChaos() error {

	client , err := delveClient.InitClient(address)
	if err != nil{
		log.Errorf("Failed to init client , error - %s" , err.Error())
		fmt.Printf("Failed to init client , error = %s\n" , err.Error())
		return err
	}
	var hacker types.ChaosInterface
	hacker , err = functionChaos.NewFunctionChaos(client , functionChaosType , getFunctionNames(functionNames) , getFunctionNames(lines))
	if err != nil{
		log.Errorf("Failed to new function chaos , error - %s", err.Error())
		fmt.Printf("Failed to new function chaos , error - %s\n", err.Error())
		return err
	}
	if err = hacker.Invade(context.Background() , duration) ; err != nil{
		log.Errorf("Failed to invade chaos , error - %s", err.Error())
		fmt.Printf("Failed to invade chaos , error - %s\n", err.Error())
		return err
	}

	log.Infof("invade function chaos success")
	fmt.Printf("invade function chaos success\n")
	return nil
}