package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
	"github.com/umi0410/ezconfig"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	TransformerWorkers []*Transformer
	UploaderWorker     Uploader
	AllWorkerWaitGroup = new(sync.WaitGroup)
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: false, ForceColors: true})
	ezconfig.LoadConfig("KHUMU", Config, []string{"./config", os.Getenv("KHUMU_CONFIG_PATH")})
}

func main() {
	logrus.Printf("KHUMU_ENVIRONMENT=%s", os.Getenv("KHUMU_ENVIRONMENT"))
	InitTaskChannels()
	StartTransformerWorkers()
	StartUploaderWorker()
	e := NewEcho()
	// echo 서버 실행
	go func() {
		err := e.Start(fmt.Sprintf("%s:%d", Config.Host, Config.Port))
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				logrus.Info("서버가 성공적으로 종료되었습니다.")
			} else {
				logrus.Error(err)
			}
		}
	}()

	// echo의 graceful shutdown 예시
	// https://echo.labstack.com/cookbook/graceful-shutdown/
	// os의 SIGINT 신호를 받은 경우 echo를 grafully shutdown한다. (현재 있는 connection은 유지)
	// echo 서버는 현재 연결되어있는 커넥션까지만 처리하고 종료되지만
	// 이미지 변환은 쌓인 작업을 모두 처리하고 전체 서비스가 종료된다.
	shutDownChan := make(chan os.Signal, 1)
	signal.Notify(shutDownChan, os.Interrupt)
	<-shutDownChan
	if err := e.Shutdown(context.Background()); err != nil {
		logrus.Error(err)
	}

	for i := 0; i < len(TransformerWorkers); i++ {
		go func(transformerIdx int) {
			logrus.Infof("Transformer %d에게 종료 신호를 보냅니다.", transformerIdx)
			TransformerWorkers[transformerIdx].Quit <- struct{}{}
		}(i)
	}

	completedAllTransformerWorkers := make(chan struct{})
	go func() {
		logrus.Info("모든 워커들이 작업을 종료했는지 확인합니다.")
		AllWorkerWaitGroup.Wait()
		logrus.Info("모든 워커들이 작업을 종료했습니다.")
		completedAllTransformerWorkers <- struct{}{}
	}()

	select {
	case <-completedAllTransformerWorkers:
		logrus.Info("모든 워커들이 작업을 종료하여 서버를 안전하게 종료합니다. 혹시 모를 미완료된 업로드 작업을 위해 5초를 대기합니다.")
		time.Sleep(5 * time.Second)
		os.Exit(0)
	case <-time.After(20 * time.Second):
		logrus.Error("Graceful shutdown의 Max timeout인 20초가 경과되었음에도 작업을 모두 완료하지 못했습니다. 강제로 종료합니다.")
		os.Exit(1)
	}
}

func StartTransformerWorkers() {
	num := Config.NumOfTransformerWorkers
	TransformerWorkers = make([]*Transformer, num)
	for i := 0; i < num; i++ {
		TransformerWorkers[i] = NewTransformer(ResizeTaskChan, ThumbnailTaskChan, UploadTaskChan, make(chan interface{}), AllWorkerWaitGroup)
		go TransformerWorkers[i].Start()
		logrus.Info("Started TransformerWorker", i)
	}
}

func StartUploaderWorker() {
	if Config.Storage.Disk.Enabled {
		UploaderWorker = &DiskUploader{
			UploadTaskChan: UploadTaskChan,
			Quit:           make(chan interface{}),
		}
	} else if Config.Storage.Aws.Enabled {
		sess, err := session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Region: aws.String("ap-northeast-2"),
			},
		})
		if err != nil {
			logrus.Fatal(err)
		}
		UploaderWorker = NewS3Uploader(UploadTaskChan, make(chan interface{}), sess, AllWorkerWaitGroup)
	} else {
		logrus.Fatal("Unsupported storage kind.")
	}
	go UploaderWorker.Start()
	logrus.Info("Started UploaderWorker. ", UploaderWorker)

}
