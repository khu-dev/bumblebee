package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stretchr/testify/assert"
	"net/http"
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
	bucketName string = "dev-khumu-disk"
)

func BeforeEachUploadTest_DiskUploader(tb testing.TB) {
	uploaderQuit = make(chan interface{}, 100)
	uploadTaskChan = make(chan *ImageUploadTask, 100)
	uploader = &DiskUploader{
		UploadTaskChan: uploadTaskChan,
		Quit:           uploaderQuit,
	}
}

func AfterEachUploadTest_DiskUploader(tb testing.TB) {
	err := os.RemoveAll(uploadPathForTest)
	assert.NoError(tb, err)
	uploaderQuit = nil
	uploadTaskChan = nil
	uploader = nil
}

func BeforeEachUploadTest_S3Uploader(tb testing.TB) {
	uploaderQuit = make(chan interface{}, 100)
	uploadTaskChan = make(chan *ImageUploadTask, 100)
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("ap-northeast-2"),
		},
	})
	assert.NoError(tb, err)
	uploader = &S3Uploader{
		UploadTaskChan: uploadTaskChan,
		Quit:           uploaderQuit,
		bucketName: bucketName,
		sess: sess,
		s3Uploader: s3manager.NewUploader(sess),
	}
}

// S3의 테스트 데이터도 지운다.
func AfterEachUploadTest_S3Uploader(tb testing.TB) {
	s3Service := s3.New(uploader.(*S3Uploader).sess)
	objects, err := s3Service.ListObjects(
		&s3.ListObjectsInput{
			Bucket: aws.String(uploader.(*S3Uploader).bucketName),
			Prefix: aws.String(uploadPathForTest),
		},
	)
	assert.NoError(tb, err)
	objectIdentifiers := make([]*s3.ObjectIdentifier, 0)
	for _, object := range objects.Contents{
		objectIdentifiers = append(objectIdentifiers, &s3.ObjectIdentifier{Key: object.Key})
	}

	_, err = s3Service.DeleteObjects(
		&s3.DeleteObjectsInput{
			Bucket: aws.String(uploader.(*S3Uploader).bucketName),
			Delete: &s3.Delete{
				Objects: objectIdentifiers,
			},
		},
	)
	assert.NoError(tb, err)

	uploaderQuit = nil
	uploadTaskChan = nil
	uploader = nil
}

func TestS3Uploader_Upload(t *testing.T) {
	BeforeEachUploadTest_S3Uploader(t)
	defer AfterEachUploadTest_S3Uploader(t)

	imageData := DownloadSampleImage(t)
	task := &ImageUploadTask{
		BaseImageTask: &BaseImageTask{
			ImageData: imageData, OriginalFileName: "google_logo.png", HashedFileName: "abcd1234abcd.png",
		},
		UploadPath: uploadPathForTest,
	}
	err := uploader.Upload(task)
	assert.NoError(t, err)

	resp, err := http.Get(fmt.Sprintf("https://dev-khumu-disk.s3.ap-northeast-2.amazonaws.com/%s/%s", task.UploadPath, task.HashedFileName))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDiskUploader_Start(t *testing.T) {
	BeforeEachUploadTest_DiskUploader(t)
	defer AfterEachUploadTest_DiskUploader(t)

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
