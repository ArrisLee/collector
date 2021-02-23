package main

import (
	"log"

	"github.com/spf13/viper"
)

func initViper() {
	viper.SetConfigName(applicationName)
	viper.AddConfigPath("./")   // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("Fatal error config file: %s \n", err)
	}
	viper.SetDefault("active_source_sftp_servers", "airgate,mmsc")
	viper.SetDefault("local_file_storage_path", "./files")
	viper.SetDefault("time_zone", "Pacific/Auckland")
	viper.SetDefault("environment", "test")
}
