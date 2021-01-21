package main

import (
	"fmt"
	"github.com/nfnt/resize"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	ThumbnailWidth  = 64
	ThumbnailHeight = 64
)

// 테스트 주도 개발시에 의존성 주입을 할 수 있도록하기 위해 Task chan을 field로도 넣음
type Transformer struct {
	ResizeTaskChan    chan *ImageResizeTask
	ThumbnailTaskChan chan *ImageGenerateThumbnailTask
	UploadTaskChan    chan *ImageUploadTask
	Quit              <-chan interface{} // 테스트 진행 시 Start를 끝내기 위함
}

// transformer가 이미지 변환작업을 시작한다.
func (t *Transformer) Start() {
	for loop := true; loop; {
		select {
		case thumbnailTask := <-t.ThumbnailTaskChan:
			t.GenerateThumbnail(thumbnailTask)
			uploadTask := &ImageUploadTask{
				BaseImageTask: thumbnailTask.BaseImageTask,
				UploadPath:    "thumbnail",
			}
			t.UploadTaskChan <- uploadTask
		case resizeTask := <-t.ResizeTaskChan:
			t.Resize(resizeTask)
			uploadTask := &ImageUploadTask{
				BaseImageTask: resizeTask.BaseImageTask,
				UploadPath:    path.Join("resized", strconv.Itoa(resizeTask.MaxWidth)),
			}
			t.UploadTaskChan <- uploadTask
		case <-time.After(1 * time.Second):
			fmt.Println("Thumbnail timeout.")
			select {
			case resizeTask := <-ResizeTaskChan:
				fmt.Println(resizeTask)
			case <-time.After(3 * time.Second):
				fmt.Println("Resize timeout.")
			}
		case <-t.Quit:
			loop = false
		}

	}
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

func ParseImageFileName(fileName string) (pureName, extension string) {
	return strings.Split(fileName, ".")[0], strings.Split(fileName, ".")[1]
}
