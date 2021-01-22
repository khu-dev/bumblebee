package main

import (
    "bytes"
    "errors"
    "fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
    "image/jpeg"
    "image/png"
    "log"
    "os"
    "path"
    "github.com/aws/aws-sdk-go/aws/session"
)

type Uploader interface {
    Start()
    Upload(task *ImageUploadTask) error
}

type S3Uploader struct {
    UploadTaskChan chan *ImageUploadTask
    Quit <-chan interface{}
    sess *session.Session
    s3Uploader *s3manager.Uploader
    bucketName string
}

type DiskUploader struct {
    UploadTaskChan chan *ImageUploadTask
    Quit           <-chan interface{} // 테스트 진행 시 Start를 끝내기 위함
}

func (u S3Uploader) Start() {
    for loop := true; loop; {
        select {
        case uploadTask := <-u.UploadTaskChan:
            u.Upload(uploadTask)
        case <-u.Quit:
            loop = true
        }
    }
}

func (u S3Uploader) Upload(task *ImageUploadTask) error {
    body := bytes.NewBuffer([]byte{})
    _, ext := ParseImageFileName(task.HashedFileName)

    switch ext {
    case "png":
        err := png.Encode(body, task.ImageData)
        if err != nil {
            log.Fatal(3, err)
        }
    case "jpg", "jpeg":
        err := jpeg.Encode(body, task.ImageData, nil)
        if err != nil {
            log.Fatal(4, err)
        }
    }

    _, err := u.s3Uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String(u.bucketName),
        Body: body,
        Key: aws.String(path.Join(task.UploadPath, task.HashedFileName)),
    })

    if err != nil{
        return err
    }

    return nil
}

func (u *DiskUploader) Start() {
    for loop := true; loop; {
        select {
        case uploadTask := <-u.UploadTaskChan:
            u.Upload(uploadTask)
        case <-u.Quit:
            loop = true
        }
    }
}

func (u *DiskUploader) Upload(task *ImageUploadTask) error {
    fmt.Println("Uploading...", task)
    fileName := task.HashedFileName
    file, err := os.Create(path.Join(task.UploadPath, fileName))
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            err := os.MkdirAll(task.UploadPath, 0755)
            file, err = os.Create(path.Join(task.UploadPath, fileName))
            if err != nil {
                log.Fatal(1, err)
            }
        } else {
            log.Fatal(2, err)
        }

    }
    _, ext := ParseImageFileName(fileName)
    switch ext {
    case "png":
        err := png.Encode(file, task.ImageData)
        if err != nil {
            log.Fatal(3, err)
        }
    case "jpg", "jpeg":
        err := jpeg.Encode(file, task.ImageData, nil)
        if err != nil {
            log.Fatal(4, err)
        }
    }

    return nil
}
