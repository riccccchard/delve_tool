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
	"delve_tool/httpServerChaos"
	"delve_tool/log"
	"delve_tool/types"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"time"
)

// httpchaosCmd represents the httpchaos command
var httpchaosCmd = &cobra.Command{
	Use:   "httpchaos",
	Short: "http chaos is used to inject chaos to http server",
	Long: `http chaos is used to inject chaos to http server , including "request_error" , "delay"`,
	Run: func(cmd *cobra.Command, args []string) {
		log.InitLog(log.InfoLvl)
		defer log.Flush()
		fmt.Println("http chaos")
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

		g.Go(func() error{
			return initAndRunHttpChaos()
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
	httpchaosType        string
	//当为delay type时，指明需要延迟的时间
	httpchaosDelay       time.Duration
)

func init() {
	rootCmd.AddCommand(httpchaosCmd)

	httpchaosCmd.Flags().StringVar(&httpchaosType , "type" , "delay" , "the http chaos type , including \"delay\" , \"request_error\" ")

	httpchaosCmd.Flags().DurationVar(&httpchaosDelay , "delay" , 500 * time.Millisecond , "delay time of delay type")

}

func initAndRunHttpChaos() error{
	client , err := delveClient.InitClient(address)
	if err != nil{
		log.Errorf("Failed to init client , error - %s" , err.Error())
		fmt.Printf("Failed to init client , error = %s\n" , err.Error())
		return err
	}
	var hacker types.ChaosInterface
	switch httpchaosType{
	case httpServerChaos.Delay_type:
		hacker , err = httpServerChaos.NewHttpServerHacker(client , httpchaosType , httpchaosDelay)

	case httpServerChaos.Request_error_type:
		hacker , err = httpServerChaos.NewHttpServerHacker(client , httpchaosType)

	default:
		log.Errorf("unknown http chaos type")
		fmt.Printf("unknown http chaos type\n")
		return errors.New("unknown http chaos type")
	}
	if err != nil{
		log.Errorf("Failed to new http chaos , error - %s", err.Error())
		fmt.Printf("Failed to new http chaos , error - %s\n", err.Error())
		return err
	}
	if err = hacker.Invade(context.Background() , duration) ; err != nil{
		log.Errorf("Failed to invade chaos , error - %s", err.Error())
		fmt.Printf("Failed to invade chaos , error - %s\n", err.Error())
		return err
	}

	log.Infof("invade http chaos success")
	fmt.Printf("invade http chaos success\n")
	return nil
}
