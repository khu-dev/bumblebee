package main

import "image"

var (
	ResizeTaskChan    chan *ImageResizeTask
	ThumbnailTaskChan chan *ImageGenerateThumbnailTask
	UploadTaskChan    chan *ImageUploadTask
)

type BaseImageTask struct {
	OriginalFileName string
	HashedFileName   string
	ImageData        image.Image
}

type ImageResizeTask struct {
	*BaseImageTask
	ResizingWidth         int
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

func InitTaskChannels(){
	ResizeTaskChan = make(chan *ImageResizeTask)
	ThumbnailTaskChan = make(chan *ImageGenerateThumbnailTask)
	UploadTaskChan = make(chan *ImageUploadTask)
}