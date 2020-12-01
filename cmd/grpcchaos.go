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
	"delve_tool/grpcServerChaos"
	"context"
	"delve_tool/types"
	"delve_tool/log"
	"fmt"
	"golang.org/x/sync/errgroup"

	"errors"

	"github.com/spf13/cobra"

	"time"
)

// grpcchaosCmd represents the grpcchaos command
var grpcchaosCmd = &cobra.Command{
	Use:   "grpcchaos",
	Short: "grpc chaos is used to inject chaos to grpc service side",
	Long: `grpc chaos is used to inject chaos to grpc service side , include "delay" , the response code chaos is coming soon`,
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

		g.Go(func() error{
			return initAndRunGrpcChaos()
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
	grpcChaosType string
	//当为delay type时，指明需要延迟的时间
	grpcDelay       time.Duration
)
func init() {
	rootCmd.AddCommand(grpcchaosCmd)

	grpcchaosCmd.Flags().StringVar(&grpcChaosType, "type" , "delay" , "the grpc chaos type , including delay")

	grpcchaosCmd.Flags().DurationVar(&grpcDelay , "delay" , 500 * time.Millisecond , "the delay time of chaos type")
}

func initAndRunGrpcChaos() error{
	client , err := delveClient.InitClient(address)
	if err != nil{
		log.Errorf("Failed to init client , error - %s" , err.Error())
		fmt.Printf("Failed to init client , error = %s\n" , err.Error())
		return err
	}
	var hacker types.ChaosInterface
	switch grpcChaosType {
	case grpcServerChaos.Delay_type:
		hacker , err = grpcServerChaos.NewgRPCChaos(client , grpcChaosType , grpcDelay)

		//TODO: 加入修改gRPC调用返回值的chaos
	case grpcServerChaos.Response_error_type:
		hacker , err = grpcServerChaos.NewgRPCChaos(client , grpcChaosType)
	default:
		log.Errorf("unknown gRPC chaos type")
		fmt.Printf("unknown gRPC chaos type\n")
		return errors.New("unknown gRPC chaos type")
	}
	if err != nil{
		log.Errorf("Failed to new gRPC chaos , error - %s", err.Error())
		fmt.Printf("Failed to new gRPC chaos , error - %s\n", err.Error())
		return err
	}
	if err = hacker.Invade(context.Background() , duration) ; err != nil{
		log.Errorf("Failed to invade chaos , error - %s", err.Error())
		fmt.Printf("Failed to invade chaos , error - %s\n", err.Error())
		return err
	}

	log.Infof("invade gRPC chaos success")
	fmt.Printf("invade gRPC chaos success\n")
	return nil
}

