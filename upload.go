package main

import (
    "bytes"
    "errors"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
    "github.com/sirupsen/logrus"
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
        bucketName: Config.Storage.Aws.BucketName,
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

    switch task.Extension {
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
        Key: aws.String(path.Join(task.UploadPath, task.HashedFileName + "." + task.Extension)),
        ContentType: aws.String("image/" + task.Extension),
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
        logrus.Error(NilImageDataErr)
        return NilImageDataErr
    }

    file, err := os.Create(path.Join(task.UploadPath, task.HashedFileName + "." + task.Extension))
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            err := os.MkdirAll(task.UploadPath, 0755)
            file, err = os.Create(path.Join(task.UploadPath, task.HashedFileName + "." + task.Extension))
            if err != nil {
                logrus.Error(err)
                return err
            }
        } else {
            logrus.Error(err)
            return err
        }

    }

    switch task.Extension {
    case "png":
        err := png.Encode(file, task.ImageData)
        if err != nil {
            log.Fatal(3, err)
        }
    case "jpeg":
        err := jpeg.Encode(file, task.ImageData, nil)
        if err != nil {
            log.Fatal(4, err)
        }
    }

    return nil
}
