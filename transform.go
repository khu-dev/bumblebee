package main

import (
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
	"path"
	"strconv"
	"sync"
	"time"
)

var (
	ThumbnailWidth  = 128
	// 새로운 사이즈의 리사이징이 필요할 경우 이곳만 바꿔주면 된다.
	ResizeSizes = []int{256, 512, 1024}
	autoIncrementTransformerID = 0
)

// 테스트 주도 개발시에 의존성 주입을 할 수 있도록하기 위해 Task chan을 field로도 넣음
type Transformer struct {
	ID int
	// 썸네일 생성 요청을 받아 처리하기 위한 채널.
	ThumbnailTaskChan <-chan *ImageGenerateThumbnailTask
	// 리사이즈 요청을 받아 처리하기 위한 채널.
	ResizeTaskChan    <-chan *ImageResizeTask
	// 이미지 변환 후 업로드하기 위한 작업 요청 채널
	UploadTaskChan    chan<- *ImageUploadTask
	// 테스트 진행 시에나 graceful shutdown을 통해 Transformer에게 종료이벤트를 전달하기 위함
	Quit              chan interface{}
	// Quit channel을 통해 quit 요청이 들어와서 quit하고자 하는 상태인지
	quit bool
	// 작업을 모두 마무리했는지 보고하기 위함.
	done *sync.WaitGroup
}

func NewTransformer(resizeChan chan *ImageResizeTask, thumbnailChan chan *ImageGenerateThumbnailTask, uploadChan chan *ImageUploadTask, quit chan interface{}, done *sync.WaitGroup) *Transformer{
	autoIncrementTransformerID++
	return &Transformer{
		ID: autoIncrementTransformerID,
		ResizeTaskChan: resizeChan,
		ThumbnailTaskChan: thumbnailChan,
		UploadTaskChan: uploadChan,
		Quit: quit,
		done: done,
	}
}

// transformer가 이미지 변환작업을 시작한다.
func (t *Transformer) Start() {
	logrus.Print("Started Transformer")
	defer logrus.Info("Finished Transformer")
	t.done.Add(1)
	defer t.done.Done()
	loop:
	for {
		logrus.Warn("Dummy wait for debugging")
		time.Sleep(3 * time.Second)
		select {
		case thumbnailTask := <-t.ThumbnailTaskChan:

			logrus.Println("ThumbnailTask", thumbnailTask)
			t.GenerateThumbnail(thumbnailTask)
			uploadTask := &ImageUploadTask{
				BaseImageTask: &BaseImageTask{
					OriginalFileName: thumbnailTask.OriginalFileName,
					HashedFileName:   thumbnailTask.HashedFileName,
					ImageData:        thumbnailTask.ThumbnailImageData,
					Extension:        thumbnailTask.Extension,
				},
				UploadPath: "thumbnail",
			}
			t.UploadTaskChan <- uploadTask
			logrus.Println("Add UploadTask", uploadTask)
		case resizeTask := <-t.ResizeTaskChan:
			logrus.Println("ResizeTask", resizeTask)
			if resizeTask.ResizingWidth < resizeTask.ImageData.Bounds().Dx() {
				t.Resize(resizeTask)
			} else {
				// Resize 필요 없음.
				resizeTask.ResizedImageData = resizeTask.ImageData
			}

			uploadTask := &ImageUploadTask{
				BaseImageTask: &BaseImageTask{
					OriginalFileName: resizeTask.OriginalFileName,
					HashedFileName:   resizeTask.HashedFileName,
					ImageData:        resizeTask.ResizedImageData,
					Extension:        resizeTask.Extension,
				},
				UploadPath: path.Join("resized", strconv.Itoa(resizeTask.ResizingWidth)),
			}
			logrus.Info("Resized image 업로드 작업 요청", uploadTask)
			t.UploadTaskChan <- uploadTask

		case <-t.Quit:
			logrus.Info("Transformer에 대한 종료 시그널이 도착했습니다.")
			t.quit = true
		default:
			if t.quit {
				logrus.Info("Transformer에 대한 종료 시그널을 받았었고, 더 이상 작업이 없습니다.")
				break loop
			} else {
				logrus.Info("처리할 작업이 없습니다. 3초 대기합니다.")
				time.Sleep(3 * time.Second)
			}
		}
	}

}

func (t *Transformer) Resize(task *ImageResizeTask) {
	w, h := t.getProperSizeBasedOnWidth(task.ResizingWidth, task.ImageData.Bounds().Dx(), task.ImageData.Bounds().Dy())
	task.ResizedImageData = resize.Resize(w, h, task.ImageData, resize.Lanczos3)
}

func (t *Transformer) GenerateThumbnail(task *ImageGenerateThumbnailTask) {
	w, h := t.getProperSizeBasedOnWidth(ThumbnailWidth, task.ImageData.Bounds().Dx(), task.ImageData.Bounds().Dy())
	task.ThumbnailImageData = resize.Resize(w, h, task.ImageData, resize.Lanczos3)
}

func (t *Transformer) getProperSizeBasedOnWidth(desiredWidth, originalW, originalH int) (uint, uint){
	// resize 안해도 됨.
	var width, height int
	if desiredWidth > originalW{
		width = originalW
		height = originalW
	} else{
		width = desiredWidth
		height = originalH * desiredWidth / originalW
	}
	return uint(width), uint(height)
}