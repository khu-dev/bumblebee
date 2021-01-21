package main

import (
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"path"
	"testing"
	"time"
)

var (
	uploader          Uploader
	uploadTaskChan    chan *ImageUploadTask
	uploaderQuit      chan interface{}
	uploadPathForTest string = "test_data_thumbnail"
)

func BeforeEachUploadTest() {
	uploaderQuit = make(chan interface{}, 100)
	uploadTaskChan = make(chan *ImageUploadTask, 100)
	uploader = &DiskUploader{
		UploadTaskChan: uploadTaskChan,
		Quit:           uploaderQuit,
	}
}

func AfterEachUploadTest() {
	uploaderQuit = nil
	uploadTaskChan = nil
	uploader = nil
	err := os.RemoveAll(uploadPathForTest)
	if err != nil {
		log.Fatal(err)
	}
}

func TestUploader_Start(t *testing.T) {
	BeforeEachUploadTest()
	defer AfterEachUploadTest()

	imageData := DownloadSampleImage(t)
	go uploader.Start()
	task := &ImageUploadTask{
		BaseImageTask: &BaseImageTask{
			ImageData: imageData, OriginalFileName: "google_logo.png", HashedFileName: "abcd1234abcd.png",
		},
		UploadPath: uploadPathForTest,
	}

	uploadTaskChan <- task
	time.Sleep(time.Second * 3)
	uploaderQuit <- struct{}{}
	_, err := os.Stat(path.Join(uploadPathForTest, task.HashedFileName))
	assert.NoError(t, err)
}
