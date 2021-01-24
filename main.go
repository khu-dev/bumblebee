package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

var (
	TransformerWorkers []*Transformer
	UploaderWorkers []Uploader
)

func init(){
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: false, ForceColors: true})
}
func main() {
	logrus.Printf("KHUMU_ENVIRONMENT=%s", os.Getenv("KHUMU_ENVIRONMENT"))
	InitConfig()
	InitTaskChannels()
	StartTransformerWorkers()
	StartUploaderWorkers()
	logrus.Fatal(NewEcho().Start(fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt("port"))))
}

func StartTransformerWorkers() {
	num := viper.GetInt("numOfTransformerWorkers")
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
	num := viper.GetInt("numOfUploaderWorkers")
	UploaderWorkers = make([]Uploader, num)
	for i := 0; i < num; i++ {
		if viper.GetBool("storage.disk.enabled") {
			UploaderWorkers[i] = &DiskUploader{
				UploadTaskChan: UploadTaskChan,
				Quit: make(chan interface{}),
			}
		} else if viper.GetBool("storage.aws.enabled"){
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
