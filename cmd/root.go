// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"os"

	"crypto/tls"
	"crypto/x509"
	"github.com/achanno/sreapi/certs"
	pb "github.com/achanno/sreapi/protobuf"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"time"
)

const (
	port = ":5555"
	host = "localhost"
	apiv = "1"
)

var (
	cfgFile      string
	demoKeyPair  *tls.Certificate
	demoCertPool *x509.CertPool
	conn         *grpc.ClientConn
	c            pb.VirtualmachinesClient
	ctx          context.Context
	cancel       context.CancelFunc
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sreapi",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer conn.Close()
	defer cancel()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sreapi.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Initialize certs
	pair, err := tls.X509KeyPair([]byte(certs.Cert), []byte(certs.Key))
	if err != nil {
		log.Fatalf("Error setting up tls: %v", err)
	}

	demoKeyPair = &pair
	demoCertPool = x509.NewCertPool()
	ok := demoCertPool.AppendCertsFromPEM([]byte(certs.Cert))
	if !ok {
		log.Fatalf("Bad cert")
	}

	// Initialize grpc connection
	creds := credentials.NewClientTLSFromCert(demoCertPool, host+port)
	conntmp, err := grpc.Dial(host+port, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Error connecting to grpc: %v", err)
	}
	conn = conntmp

	// Initialize context
	c = pb.NewVirtualmachinesClient(conn)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)

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

		// Search config in home directory with name ".sreapi" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".sreapi")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
