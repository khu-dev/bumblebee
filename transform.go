package main

import (
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
	"path"
	"strconv"
)

var (
	ThumbnailWidth  = 64
	ThumbnailHeight = 64
	autoIncrementTransformerID = 0
)

// 테스트 주도 개발시에 의존성 주입을 할 수 있도록하기 위해 Task chan을 field로도 넣음
type Transformer struct {
	ID int
	ResizeTaskChan    chan *ImageResizeTask
	ThumbnailTaskChan chan *ImageGenerateThumbnailTask
	UploadTaskChan    chan *ImageUploadTask
	Quit              <-chan interface{} // 테스트 진행 시 Start를 끝내기 위함
}

func NewTransformer(resizeChan chan *ImageResizeTask, thumbnailChan chan *ImageGenerateThumbnailTask, uploadChan chan *ImageUploadTask, quit chan interface{}) *Transformer{
	autoIncrementTransformerID++
	return &Transformer{
		ID: autoIncrementTransformerID,
		ResizeTaskChan: resizeChan,
		ThumbnailTaskChan: thumbnailChan,
		UploadTaskChan: uploadChan,
		Quit: quit,
	}
}

// transformer가 이미지 변환작업을 시작한다.
func (t *Transformer) Start() {
	logrus.Print("Started Transformer")
	for loop := true; loop; {
		select {
		case thumbnailTask := <-t.ThumbnailTaskChan:
			logrus.Println("ThumbnailTask", thumbnailTask)
			t.GenerateThumbnail(thumbnailTask)
			uploadTask := &ImageUploadTask{
				BaseImageTask: &BaseImageTask{
					OriginalFileName: thumbnailTask.OriginalFileName,
					HashedFileName: thumbnailTask.HashedFileName,
					ImageData: thumbnailTask.ThumbnailImageData,
				},
				UploadPath:    "thumbnail",
			}
			t.UploadTaskChan <- uploadTask
			logrus.Println("Add UploadTask", uploadTask)
		case resizeTask := <-t.ResizeTaskChan:
			logrus.Println("ResizeTask", resizeTask)
			t.Resize(resizeTask)
			uploadTask := &ImageUploadTask{
				BaseImageTask: &BaseImageTask{
					OriginalFileName: resizeTask.OriginalFileName,
					HashedFileName: resizeTask.HashedFileName,
					ImageData: resizeTask.ResizedImageData,
				},
				UploadPath:    path.Join("resized", strconv.Itoa(resizeTask.MaxWidth)),
			}
			t.UploadTaskChan <- uploadTask
			logrus.Println("Add UploadTask", uploadTask)
		case <-t.Quit:
			loop = false
		}

	}
	logrus.Print("Finished Transformer")
}

func (t *Transformer) Resize(task *ImageResizeTask) {
	task.ResizedImageData = resize.Resize(uint(task.MaxWidth), uint(task.MaxHeight), task.ImageData, resize.Lanczos3)
	task.ImageData = nil // 이제 필요 없으니 지워줘서 GC가 처리할 수 있게 함.
	//_, imageExtensionName := ParseImageFileName(task.OriginalFileName)
	//if imageExtensionName == "png"{
	//    task.ImageData = resize.Resize(uint(task.MaxWidth), uint(task.MaxHeight), task.ImageData, resize.Lanczos3)
	//}
}

func (t *Transformer) GenerateThumbnail(task *ImageGenerateThumbnailTask) {
	task.ThumbnailImageData = resize.Resize(uint(ThumbnailWidth), uint(ThumbnailHeight), task.ImageData, resize.Lanczos3)
	task.ImageData = nil
}

