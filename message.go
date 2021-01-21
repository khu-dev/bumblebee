package main

import "image"

var(
    ResizeTaskChan chan *ImageResizeTask
    ThumbnailTaskChan chan *ImageGenerateThumbnailTask
    UploadTaskChan chan *ImageUploadTask
)

type BaseImageTask struct {
    OriginalFileName string
    HashedFileName string
    ImageData image.Image
}

type ImageResizeTask struct {
    *BaseImageTask
    MaxWidth int
    MaxHeight int
    ResizedImageData image.Image
}

type ImageGenerateThumbnailTask struct{
    *BaseImageTask
    ThumbnailImageData image.Image
}

type ImageUploadTask struct{
    *BaseImageTask
    UploadPath string
}