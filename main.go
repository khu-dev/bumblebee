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
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	// 이미지 변환 작업 요청을 처리하는 워커
	TransformerWorkers []*Transformer
	// 업로드 작업 요청을 처리하는 워커
	UploaderWorker     Uploader
	// TransformerWorkers의 작업이 모두 마무리되었는지를 관리하는 WaitGroup
	transformerWorkersWG = new(sync.WaitGroup)
)

func init() {
	workingDir, err := os.Getwd()
	if err != nil {
		logrus.Error(err)
	}

	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		DisableQuote:  true,
		ForceColors:   true,
		// line을 깔끔하게 보여줌.
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := strings.Replace(f.File, workingDir+"/", "", -1)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
		FullTimestamp:   false,
		TimestampFormat: "2006/01/03 15:04:05",
	})
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

	// Transformer들에게 SIGINT가 발생했음을 전파
	for i := 0; i < len(TransformerWorkers); i++ {
		go func(transformerIdx int) {
			logrus.Infof("Transformer %d에게 종료 신호를 보냅니다.", transformerIdx)
			TransformerWorkers[transformerIdx].Quit <- struct{}{}
		}(i)
	}

	select {
	case <-allTransformerWorkersCompleted():
		logrus.Infof("모든 워커들이 작업을 종료하여 서버를 안전하게 종료합니다. 혹시 모를 미완료된 업로드 작업을 위해 %d초를 대기합니다.", Config.GracefulShutdown.UploaderTimeout)
		time.Sleep(time.Duration(Config.GracefulShutdown.UploaderTimeout) * time.Second)
		os.Exit(0)
	case <-time.After(time.Duration(Config.GracefulShutdown.MaxTimeout) * time.Second):
		logrus.Errorf("Graceful shutdown의 Max timeout인 %d초가 경과되었음에도 작업을 모두 완료하지 못했습니다. 강제로 종료합니다. (개발 환경에서 편의상 transformer의 loop delay보다 짧게 기다리는 경우 발생할 수도 있음.)", Config.GracefulShutdown.MaxTimeout)
		os.Exit(1)
	}
}

func StartTransformerWorkers() {
	num := Config.NumOfTransformerWorkers
	TransformerWorkers = make([]*Transformer, num)
	for i := 0; i < num; i++ {
		TransformerWorkers[i] = NewTransformer(ResizeTaskChan, ThumbnailTaskChan, UploadTaskChan, make(chan interface{}), transformerWorkersWG)
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
		UploaderWorker = NewS3Uploader(UploadTaskChan, make(chan interface{}), sess)
	} else {
		logrus.Fatal("Unsupported storage kind.")
	}
	go UploaderWorker.Start()
	logrus.Info("Started UploaderWorker. ", UploaderWorker)

}

// 모든 Transformer가 작업을 종료하면 channel에 값을 전달합니다.
// channel을 return하기 때문에 select문을 통해 timeout이나 default를 이용하기 편라합니다.
func allTransformerWorkersCompleted() <-chan struct{}{
	logrus.Info("모든 워커들이 작업을 종료했는지 확인합니다.")
	isAllCompleted := make(chan struct{})
	go func() {

		transformerWorkersWG.Wait()
		logrus.Info("모든 워커들이 작업을 종료했습니다.")
		isAllCompleted <- struct{}{}
	}()
	return isAllCompleted
}
