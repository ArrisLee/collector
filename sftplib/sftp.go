package sftplib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

//Client - sftp client obj
type Client struct {
	host, user, password string
	port                 int
	*sftp.Client
}

// NewConn creates a new SFTP connection by given parameters
func NewConn(host, user, password string, port int) (client *Client, err error) {
	switch {
	case strings.TrimSpace(host) == "", strings.TrimSpace(user) == "", strings.TrimSpace(password) == "", port <= 0 || port > 65535:
		return nil, errors.New("Invalid SFTP config parameters")
	}

	client = &Client{
		host:     host,
		user:     user,
		password: password,
		port:     port,
	}

	if err = client.connect(); nil != err {
		return nil, err
	}
	return client, nil
}

func (sc *Client) connect() error {
	config := &ssh.ClientConfig{
		User:            sc.user,
		Auth:            []ssh.AuthMethod{ssh.Password(sc.password)},
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// connet to ssh
	addr := fmt.Sprintf("%s:%d", sc.host, sc.port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	// create sftp client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	sc.Client = client
	return nil
}

// UploadFile uploads a single file to sftp server
func (sc *Client) UploadFile(localFile, remoteFile string) error {
	srcFile, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	// Make remote directories recursion
	parent := filepath.Dir(remoteFile)
	path := string(filepath.Separator)
	dirs := strings.Split(parent, path)
	for _, dir := range dirs {
		path = filepath.Join(path, dir)
		sc.Mkdir(path)
	}

	dstFile, err := sc.OpenFile(remoteFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}

// DownloadFile downloads a single file from sftp server
func (sc *Client) DownloadFile(remoteFile, importFile string, rawFiles []string, deleteOnDownload bool) error {
	//open remote file
	srcFile, err := sc.Open(remoteFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	//init GCP storage client
	ctx := context.Background()
	gclient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("GCP storage.NewClient, err: %v", err)
	}
	defer gclient.Close()
	bucket := gclient.Bucket(viper.GetString("google_storage_bucket_name"))

	//save import/renamed files to the bucket
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	dstImport := bucket.Object(importFile)
	imtFileWriter := dstImport.NewWriter(ctx)
	defer imtFileWriter.Close()
	if _, err := io.Copy(imtFileWriter, srcFile); err != nil {
		return fmt.Errorf("Failed to save to bucket, err: %v", err)
	}

	//save raw file copies to the raw file dir(s)
	for i := range rawFiles {
		dstRaw := bucket.Object(rawFiles[i])
		if _, err := dstRaw.CopierFrom(dstImport).Run(ctx); err != nil {
			return fmt.Errorf("Failed to save raw file copy %s, err: %v", rawFiles[i], err)
		}
	}

	if deleteOnDownload {
		go sc.DeleteFile(remoteFile)
	}
	return nil
}

// DownloadFiles downloads all files from a dir on sftp server
func (sc *Client) DownloadFiles(remotePath, bucketImportDir, bucketRawDirs, regex, newFileNamePrefix string, deleteOnDownload bool) error {
	fileInfo, err := sc.ReadDir(remotePath)
	if err != nil {
		return err
	}
	for i := range fileInfo {
		matched := false
		if regex != "" {
			matched, _ = regexp.MatchString(regex, fileInfo[i].Name())
		} else {
			matched = true
		}
		if !matched {
			continue
		}
		remoteFile := remotePath + string(filepath.Separator) + fileInfo[i].Name()
		importFile := bucketImportDir + string(filepath.Separator) + fileInfo[i].Name()
		if newFileNamePrefix != "" {
			//rename file when downloading if necessary
			importFile = bucketImportDir + string(filepath.Separator) + regenFileName(newFileNamePrefix, fileInfo[i].Name())
		}
		var rawFiles []string
		for i := range strings.Split(bucketRawDirs, ",") {
			rawFile := strings.Split(bucketRawDirs, ",")[i] + string(filepath.Separator) + fileInfo[i].Name()
			rawFiles = append(rawFiles, rawFile)
		}
		if err := sc.DownloadFile(remoteFile, importFile, rawFiles, deleteOnDownload); err != nil {
			//log the file when download failed
			log.Printf("Failed to perform file collecting for remote file: %s, err: %v", remoteFile, err)
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// DeleteFile deletes file from sftp server remotely
func (sc *Client) DeleteFile(remoteFile string) error {
	if err := sc.Remove(remoteFile); err != nil {
		return err
	}
	return nil
}

func regenFileName(prefix string, fileName string) string {
	ext := strings.Split(fileName, ".")[1]
	loc, _ := time.LoadLocation(viper.GetString("time_zone"))
	currentTime := time.Now().In(loc)
	// new := currentTime.Format("20060102150405") + fmt.Sprintf("%d", currentTime.Nanosecond())
	new := currentTime.Format("20060102150405")
	return prefix + "_" + new + "." + ext
}
