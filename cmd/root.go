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
	"delve_tool/delveServer"
	"flag"
	"fmt"
	"github.com/go-delve/delve/pkg/logflags"
	"github.com/spf13/cobra"
	"os"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"delve_tool/log"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "delve_tool",
	Short: "delve_tool is used to inject error to golang application in runtime",
	Long: `delve_tool is used to inject error to golang application in runtime. 
it uses delve to set breakpoint and modify variables at runtime , which means
you can use it to debug or inject some error to your application , such as http request error , 
gRPC request delay and so on.
At present, the delve tool is still in the experimental stage, 
and it is hoped that more elaborate fault injection scenarios and schemes can be completed in the future.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) {
	//		fmt.Printf("testing\n")
	//	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	//attach target process pid，必填
	pid         int
	//address which delve server listen to
	address     string
	//debug info print or not
	debug       bool
	//chaos experiment's duraion
	duration    time.Duration


	myDelveServer *delveServer.DelveServer

)
func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.delve_tool.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	//一些通用的参数项
	//pid必填
	rootCmd.PersistentFlags().IntVar(&pid , "pid", 0 , "attach target process pid")
	//address可选，如果不填则随机端口
	rootCmd.PersistentFlags().StringVar(&address , "address" , "127.0.0.1:0" , "address which delve server listen to")
	//debug 信息，如果为true则打印到日志文件delve_debug.log中
	rootCmd.PersistentFlags().BoolVar(&debug , "debug" , false , "whether print debug informaion , if true, information will print into delve_debug.log ")
	//duration ，实验持续时长
	rootCmd.PersistentFlags().DurationVar(&duration , "duration" , 30 * time.Second , "chaos experiment's duration")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".delve_tool" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".delve_tool")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

//启动delve server attach目标进程，waitForStopServer是阻塞的，需要起协程
func AttachTargetProcess(pid uint32, address string) error {
	myDelveServer = &delveServer.DelveServer{}
	fmt.Printf(" Initing Server ... \n")
	err := myDelveServer.InitServer(int(pid), address, duration+500*time.Millisecond) //比客户端多等0.5秒
	if err != nil {
		return err
	}
	fmt.Printf(" Staring Server ... \n")
	err = myDelveServer.StartServer()
	if err != nil {
		return err
	}
	fmt.Printf(" Waiting Server to stop... \n")
	err = myDelveServer.WaitForStopServer()
	if err != nil {
		return err
	}
	return nil
}


//打开delve server调试信息
func setupDelveServerDebugLog() {
	logflags.Setup(true, "debugger", "")
}

func checkoutArguementCorrect() bool {
	if pid <= 0 {
		fmt.Printf("pid must be Positive number!\n")
		log.Errorf("pid must be Positive number!")
		flag.Usage()
		return false
	}
	if duration <= 0 {
		fmt.Printf("duration is a negative integer , force it to 10 seconds.\n")
		log.Infof("duration is a negative integer , force it to 10 seconds.")
		duration = 10 * time.Second
	}
	return true
}

func printMainArgs(){
	log.Infof("Get main args from command , pid : %d , address : %s , duration : %v , debug : %v ", pid , address , duration , debug )
	fmt.Printf("Get main args from command , pid : %d , address : %s , duration : %v , debug : %v \n", pid , address , duration , debug )
}




