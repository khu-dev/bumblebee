package main

import (
    "log"
    "os"
    "testing"
    "time"
)

var (
    uploader Uploader
    uploadTaskChan chan *ImageUploadTask
    uploaderQuit chan interface{}
    uploadPathForTest string = "test_thumbnail"
)

func BeforeEachUploadTest() {
    uploaderQuit = make(chan interface{}, 100)
    uploadTaskChan = make(chan *ImageUploadTask, 100)
    uploader = &DiskUploader{
        UploadTaskChan: uploadTaskChan,
        Quit: uploaderQuit,
    }
}

func AfterEachUploadTest(){
    uploaderQuit = nil
    uploadTaskChan = nil
    uploader = nil
    err := os.RemoveAll(uploadPathForTest)
    if err != nil{
        log.Fatal(err)
    }
}

func TestUploader_Start(t *testing.T) {
    BeforeEachUploadTest()
    defer AfterEachUploadTest()

    imageData := DownloadSampleImage(t)
    go uploader.Start()
    uploadTaskChan <- &ImageUploadTask{
        BaseImageTask:&BaseImageTask{
            ImageData: imageData, OriginalFileName: "google_logo.png", HashedFileName: "abcd1234abcd.png",
        },
        UploadPath: uploadPathForTest,
    }
    time.Sleep(time.Second*5)
    uploaderQuit <- struct{}{}
}
