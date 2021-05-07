package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
	"github.com/umi0410/ezconfig"
	"os"
)

var (
	TransformerWorkers []*Transformer
	UploaderWorkers []Uploader
)

func init(){
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: false, ForceColors: true})
	ezconfig.LoadConfig("KHUMU", Config, []string{"./config", os.Getenv("KHUMU_CONFIG_PATH")})
}
func main() {
	logrus.Printf("KHUMU_ENVIRONMENT=%s", os.Getenv("KHUMU_ENVIRONMENT"))
	InitTaskChannels()
	StartTransformerWorkers()
	StartUploaderWorkers()
	logrus.Fatal(NewEcho().Start(fmt.Sprintf("%s:%d", Config.Host, Config.Port)))
}

func StartTransformerWorkers() {
	num := Config.NumOfTransformerWorkers
	TransformerWorkers = make([]*Transformer, num)
	for i := 0; i < num; i++ {
		TransformerWorkers[i] = NewTransformer(ResizeTaskChan, ThumbnailTaskChan, UploadTaskChan,make(chan interface{}))
		go TransformerWorkers[i].Start()
		logrus.Print("Started TransformerWorker", i)
	}
}

func StartUploaderWorkers() {
	// 대체로 UploaderWorker는 한 개만 있어도 됨.
	// 하나의 UploaderWorker가 작업이 들어오는 족족 goroutine을 실행시키기때문.
	num := Config.NumOfUploaderWorkers
	UploaderWorkers = make([]Uploader, num)
	for i := 0; i < num; i++ {
		if Config.Storage.Disk.Enabled {
			UploaderWorkers[i] = &DiskUploader{
				UploadTaskChan: UploadTaskChan,
				Quit: make(chan interface{}),
			}
		} else if Config.Storage.Aws.Enabled{
			sess, err := session.NewSessionWithOptions(session.Options{
				Config: aws.Config{
					Region: aws.String("ap-northeast-2"),
				},
			})
			if err != nil{
				logrus.Fatal(err)
			}
			UploaderWorkers[i] = NewS3Uploader(UploadTaskChan, make(chan interface{}), sess)
		} else {
			logrus.Fatal("Unsupported storage kind.")
		}
		go UploaderWorkers[i].Start()
		logrus.Print("Started UploaderWorker", i)
	}
}
