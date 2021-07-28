package main

import (
	"fmt"
	"image"
)

var (
	ResizeTaskChan    chan *ImageResizeTask
	ThumbnailTaskChan chan *ImageGenerateThumbnailTask
	UploadTaskChan    chan *ImageUploadTask
)

type BaseImageTask struct {
	// 원본 파일 이름 전체 (e.g. abcde.jpeg)
	OriginalFileName string
	// OriginalFileName을 hasing한 이름 (e.g. a1b2c3d4e5)
	HashedFileName string
	ImageData      image.Image
	// 이미지 파일 확장자명 (e.g. jpeg, png)
	Extension string
}

type ImageResizeTask struct {
	*BaseImageTask
	ResizingWidth int
	//MaxHeight        int // Height는 Width에 따라 정함.
	ResizedImageData image.Image
}

type ImageGenerateThumbnailTask struct {
	*BaseImageTask
	ThumbnailImageData image.Image
}

type ImageUploadTask struct {
	*BaseImageTask
	UploadPath string
}

func InitTaskChannels() {
	ResizeTaskChan = make(chan *ImageResizeTask)
	ThumbnailTaskChan = make(chan *ImageGenerateThumbnailTask)
	UploadTaskChan = make(chan *ImageUploadTask)
}

func (t *BaseImageTask) String() string{
	return fmt.Sprintf("BaseImageTask(OriginalFileName: %s, HashedFileName: %s)", t.OriginalFileName, t.HashedFileName)
}

func (t *ImageUploadTask) String() string{
	return fmt.Sprintf("ImageUploadTask(UploadPath: %s, OriginalFileName: %s, HashedFileName: %s)", t.UploadPath, t.OriginalFileName, t.HashedFileName)
}
