package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

//var (
//	Config *BumblebeeConfig
//)
//
//type BumblebeeConfig struct {
//	S3BucketName            string
//	NumOfTransformerWorkers int
//	NumOfUploaderWorkers int
//}

func InitConfig() {
	viper.SetEnvPrefix("khumu") // startswith KHUMU_
	environment := os.Getenv("KHUMU_ENVIRONMENT")
	switch environment{
	case "default", "dev", "":
		if environment == ""{
			viper.SetConfigName("default")
		} else {viper.SetConfigName(environment)}
	default: logrus.Fatal("Unsupported KHUMU_ENVIRONMENT.")
	}
	viper.AddConfigPath(".")               // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}
