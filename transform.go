package main

import (
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
	"path"
	"strconv"
)

var (
	ThumbnailWidth  = 128
	// 새로운 사이즈의 리사이징이 필요할 경우 이곳만 바꿔주면 된다.
	ResizeSizes = []int{256, 1024}
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
			if resizeTask.ResizingWidth < resizeTask.ImageData.Bounds().Dx(){
				t.Resize(resizeTask)
			}else{
				// Resize 필요 없음.
				resizeTask.ResizedImageData = resizeTask.ImageData
				resizeTask.ImageData = nil
			}

			uploadTask := &ImageUploadTask{
				BaseImageTask: &BaseImageTask{
					OriginalFileName: resizeTask.OriginalFileName,
					HashedFileName: resizeTask.HashedFileName,
					ImageData: resizeTask.ResizedImageData,
				},
				UploadPath:    path.Join("resized", strconv.Itoa(resizeTask.ResizingWidth)),
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
	w, h := t.getProperSizeBasedOnWidth(task.ResizingWidth, task.ImageData.Bounds().Dx(), task.ImageData.Bounds().Dy())
	task.ResizedImageData = resize.Resize(w, h, task.ImageData, resize.Lanczos3)
	task.ImageData = nil // 이제 필요 없으니 지워줘서 GC가 처리할 수 있게 함.
}

func (t *Transformer) GenerateThumbnail(task *ImageGenerateThumbnailTask) {
	w, h := t.getProperSizeBasedOnWidth(ThumbnailWidth, task.ImageData.Bounds().Dx(), task.ImageData.Bounds().Dy())
	task.ThumbnailImageData = resize.Resize(w, h, task.ImageData, resize.Lanczos3)
	task.ImageData = nil
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