/**
Channel이 메시지의 형태로 Task를 주고받습니다.
 */
package main

import (
	"errors"
	"fmt"
	"image"
	"image/gif"
)

var (
	ResizeTaskChan    chan *ImageResizeTask
	ThumbnailTaskChan chan *ImageGenerateThumbnailTask
	UploadTaskChan    chan *ImageUploadTask

	ErrNoImageErr = errors.New("이미지 데이터가 nil입니다 ImageData혹은 GIFImageData 중 적어도 하나는 데이터가 있어야합니다")
)

type BaseImageTask struct {
	// 원본 파일 이름 전체 (e.g. abcde.jpeg)
	OriginalFileName string
	// OriginalFileName을 hasing한 이름 (e.g. a1b2c3d4e5)
	HashedFileName string
	// 일반 이미지 jpeg, png, bmp
	ImageData      image.Image
	// gif는 연속적인 image로 구성됨
	GIFImageData *gif.GIF
	// 이미지 파일 확장자명 (e.g. jpeg, png)
	Extension string
}

type ImageResizeTask struct {
	*BaseImageTask
	ResizingWidth int
	//MaxHeight        int // Height는 Width에 따라 정함.
	ResizedImageData image.Image
	ResizedGIFImageData *gif.GIF
}

type ImageGenerateThumbnailTask struct {
	*BaseImageTask
	ThumbnailImageData image.Image
	ThumbnailGIFImageData *gif.GIF
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

func (t *BaseImageTask) Validate() error{
	if t.ImageData == nil && t.GIFImageData == nil {
		return ErrNoImageErr
	}

	return nil
}

func (t *BaseImageTask) GetOriginalWidth() (int, error){
	if t.ImageData != nil {
		return t.ImageData.Bounds().Dx(), nil
	} else if t.GIFImageData != nil {
		return t.GIFImageData.Config.Width, nil
	} else{
		return 0, ErrNoImageErr
	}
}

func (t *BaseImageTask) GetOriginalHeight() (int, error){
	if t.ImageData != nil {
		return t.ImageData.Bounds().Dy(), nil
	} else if t.GIFImageData != nil {
		return t.GIFImageData.Config.Height, nil
	} else{
		return 0, ErrNoImageErr
	}
}