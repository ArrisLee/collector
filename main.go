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
	sourceServerList := strings.Split(viper.GetString("active_source_sftp_servers"), ",")
	wg := &sync.WaitGroup{}
	for _, serverName := range sourceServerList {
		wg.Add(1)
		go runCollector(serverName, viper.GetString("google_storage_bucket_name"), wg)
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
	//STEP 1: load configs for each server
	sftpAddress := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.sftp_host", serverName))
	sftpUser := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.sftp_user", serverName))
	sftpPass := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.sftp_password", serverName))
	sftpPort := viper.GetInt(fmt.Sprintf("source_sftp_servers.%s.sftp_port", serverName))
	importFileDir := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.gbucket_import_file_dir", serverName))
	rawFileDir := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.gbucket_raw_file_dir", serverName))
	srcFileDir := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.source_file_dir", serverName))
	srcFileNameRegex := viper.GetString(fmt.Sprintf("source_sftp_servers.%s.filename_regex", serverName))
	renameFilesOnDownload := viper.GetBool(fmt.Sprintf("source_sftp_servers.%s.rename_files_on_download", serverName))
	deleteFilesAfterDownlaod := viper.GetBool(fmt.Sprintf("source_sftp_servers.%s.delete_files_after_download", serverName))
	collector := &collector{}
	//STEP 2: establish SFTP connection
	if err := collector.Init(sftpAddress, sftpUser, sftpPass, sftpPort); err != nil {
		log.Printf("Failed to init collector for: %s, err: %v", serverName, err)
		return
	}
	//STEP 3: config collector run params
	if err := collector.Config(srcFileDir, importFileDir, rawFileDir, srcFileNameRegex, serverName, renameFilesOnDownload, deleteFilesAfterDownlaod); err != nil {
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
	sc.UploadFile("samples_files/sample_srv_one.txt", "/srvone_remote/2021_01_01_00_00_01_00001.txt")
	sc.UploadFile("samples_files/sample_srv_one.txt", "/srvone_remote/2020_12_31_23_59_59_10000.txt")
	sc.UploadFile("samples_files/sample_srv_one.txt", "/srvone_remote/chaos_monkey_001.txt")
	sc.UploadFile("samples_files/sample_srv_two.txt", "/srvtwo_remote/F202012300001001.txt")
	sc.UploadFile("samples_files/sample_srv_two.txt", "/srvtwo_remote/F202102192359666.txt")
	sc.UploadFile("samples_files/sample_srv_two.txt", "/srvtwo_remote/chaosmonkey0230103.txt")
}
