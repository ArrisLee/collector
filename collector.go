package main

import (
	"errors"

	"github.com/AriesLee/collector/sftplib"
)

type collector struct {
	Client                *sftplib.Client
	SourceFileDir         string
	SoureFileNameRegex    string
	GBucketImportFileDir  string
	GBucketRawFileDirs    string
	RenameDownloadedFiles bool
	NewFileNamePrefix     string
	DeleteSourceFiles     bool
}

func (c *collector) Init(sourceSFTPAddr, sourceSFTPUser, sourceSFTPPass string, port int) error {
	sc, err := sftplib.NewConn(sourceSFTPAddr, sourceSFTPUser, sourceSFTPPass, port)
	if err != nil {
		return err
	}
	c.Client = sc
	return nil
}

func (c *collector) Config(sourceFileDir, importFileDir, rawFileDirs, soureFileNameRegex, newFileNamePrefix string, renameDownloadedFiles, deleteSourceFiles bool) error {
	if c == nil || c.Client == nil {
		return errors.New("Nil collector or SFTP client obj")
	}
	if sourceFileDir == "" {
		c.Client.Close()
		return errors.New("Invalid source file download directory")
	}
	if importFileDir == "" {
		c.Client.Close()
		return errors.New("Invalid Google bucket import file directory")
	}
	if renameDownloadedFiles && newFileNamePrefix == "" {
		c.Client.Close()
		return errors.New("Invalid new file name prefix")
	}
	c.SourceFileDir = sourceFileDir
	c.GBucketImportFileDir = importFileDir
	if rawFileDirs != "" {
		c.GBucketRawFileDirs = rawFileDirs
	}
	c.RenameDownloadedFiles = renameDownloadedFiles
	if c.RenameDownloadedFiles {
		c.NewFileNamePrefix = newFileNamePrefix
	}
	if soureFileNameRegex != "" {
		c.SoureFileNameRegex = soureFileNameRegex
	}
	c.DeleteSourceFiles = deleteSourceFiles
	return nil
}

func (c *collector) Run() error {
	defer c.Client.Close()
	if err := c.Client.DownloadFiles(c.SourceFileDir, c.GBucketImportFileDir, c.GBucketRawFileDirs, c.SoureFileNameRegex, c.NewFileNamePrefix, c.DeleteSourceFiles); err != nil {
		return err
	}
	return nil
}
