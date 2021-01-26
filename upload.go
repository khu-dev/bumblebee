package main

import (
    "bytes"
    "errors"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
    "github.com/sirupsen/logrus"
    "github.com/spf13/viper"
    "image/jpeg"
    "image/png"
    "log"
    "os"
    "path"
)

var (
    NilImageDataErr = errors.New("ImageData가 nil이기 때문에 업로드할 수 없습니다.")
	autoIncrementUploaderID = 0
)
type Uploader interface {
    Start()
    Upload(task *ImageUploadTask) error
}

type S3Uploader struct {
    ID int
    UploadTaskChan chan *ImageUploadTask
    Quit <-chan interface{}
    sess *session.Session
    s3Uploader *s3manager.Uploader
    bucketName string
}

type DiskUploader struct {
    ID int
    UploadTaskChan chan *ImageUploadTask
    Quit           <-chan interface{} // 테스트 진행 시 Start를 끝내기 위함

}

func NewS3Uploader(taskChan chan *ImageUploadTask, quit chan interface{}, sess *session.Session) Uploader{
    autoIncrementUploaderID++
    return &S3Uploader{
        ID: autoIncrementUploaderID,
        UploadTaskChan: taskChan,
        Quit: quit,
        bucketName: viper.GetString("storage.aws.bucketName"),
        sess: sess,
        s3Uploader: s3manager.NewUploader(sess),
    }
}

func (u S3Uploader) Start() {
    logrus.Print("Started S3Uploader")
    for loop := true; loop; {
        select {
        case uploadTask := <-u.UploadTaskChan:
            logrus.Println("Start uploading", uploadTask)
            u.Upload(uploadTask)
            logrus.Println("Finish uploading", uploadTask)
        case <-u.Quit:
            loop = true
        }
    }
    logrus.Print("Finished S3Uploader")
}

func (u S3Uploader) Upload(task *ImageUploadTask) error {
    body := bytes.NewBuffer([]byte{})
    _, ext, _ := ParseImageFileName(task.HashedFileName)

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
        ContentType: aws.String("image/" + ext),
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
    logrus.Println("Uploading...", task)
    if task.ImageData == nil{
        return NilImageDataErr
    }
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
    _, ext, _ := ParseImageFileName(fileName)
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
