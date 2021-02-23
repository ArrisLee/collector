package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/AriesLee/collector/sftplib"
	"github.com/spf13/viper"
)

var applicationName = "collector"

func main() {
	initViper()
	warmup()
	//STEP 1: read to-be-collected server list
	sourceServerList := strings.Split(viper.GetString("active_source_sftp_servers"), ",")
	wg := &sync.WaitGroup{}
	for _, serverName := range sourceServerList {
		wg.Add(1)
		go runCollector(serverName, localFileStoragePath, wg)
	}
	wg.Wait()
}

func warmup() {
	switch {
	case viper.GetString("active_source_sftp_servers") == "":
		log.Fatal("Empty sftp list")
	case viper.GetString("local_file_storage_path") == "":
		log.Fatal("Empty local file storage path")
	case viper.GetString("google_storage_bucket_name") == "":
		log.Fatal("Empty Google storage bucket name")
	case viper.GetString("google_auth_json_location") != "":
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", viper.GetString("google_auth_json_location"))
	case viper.GetString("environment") == "test" && viper.GetBool("upload_sample_testing_files"):
		uploadSampleFiles()
	}
}

func runCollector(serverName, localFileStoragePath string, wg *sync.WaitGroup) {
	defer wg.Done()
	//STEP 2: read configs for each server
	sftpAddress := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.sftp_host", serverName))
	sftpUser := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.sftp_user", serverName))
	sftpPass := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.sftp_password", serverName))
	sftpPort := viper.GetInt(fmt.Sprintf("source_sftp_servers.%s.sftp_port", serverName))
	localFileDir := localFileStoragePath + viper.GetString(fmt.Sprintf("source_sftp_servers.%s.local_file_dir", serverName))
	srcFileDir := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.source_file_dir", serverName))
	srcFileNameRegex := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.filename_regex", serverName))
	renameFilesOnDownload := viper.GetBool(fmt.Sprintf("source_sftp_servers.%s.rename_files_on_download", serverName))
	deleteFilesAfterDownlaod := viper.GetBool(fmt.Sprintf("source_sftp_servers.%s.delete_files_after_download", serverName))
	if _, err := os.Stat(localFileDir); os.IsNotExist(err) {
		os.MkdirAll(localFileDir, 0700)
	}
	collector := &collector{}
	//STEP 3: establish SFTP connection
	if err := collector.Init(sftpAddress, sftpUser, sftpPass, sftpPort); err != nil {
		log.Printf("Failed to init collector for: %s, err: %v", serverName, err)
		return
	}
	//STEP 4: config SFTP file download params
	if err := collector.Config(srcFileDir, localFileDir, srcFileNameRegex, serverName, renameFilesOnDownload, deleteFilesAfterDownlaod); err != nil {
		log.Printf("Failed to configure collector for: %s, err: %v", serverName, err)
		return
	}
	//STEP 5: run the collector
	log.Println(serverName + " collector started...")
	if err := collector.Run(); err != nil {
		log.Printf("Failed to run collector for: %s, err: %v", serverName, err)
	}
}

func uploadSampleFiles() {
	sc, _ := sftplib.NewConn(viper.GetString("test_sftp_host"),
		viper.GetString("test_sftp_user"),
		viper.GetString("test_sftp_password"),
		viper.GetInt("test_sftp_port"))
	sc.UploadFile("samples/airgate.cdr", "/airgate_remote/2021_01_01_00_00_01_00001.cdr")
	sc.UploadFile("samples/airgate.cdr", "/airgate_remote/2020_12_31_23_59_59_10000.cdr")
	sc.UploadFile("samples/airgate.cdr", "/airgate_remote/chaos_monkey_001.cdr")
	sc.UploadFile("samples/mmsc.ic", "/mmsc_remote/F202012300001001.ic")
	sc.UploadFile("samples/mmsc.ic", "/mmsc_remote/F202102192359666.ic")
	sc.UploadFile("samples/mmsc.ic", "/mmsc_remote/chaosmonkey0230103.ic")
}
