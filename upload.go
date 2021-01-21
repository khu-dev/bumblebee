package main

import (
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path"
)

type Uploader interface {
	Start()
	Upload(task *ImageUploadTask) error
}

type DiskUploader struct {
	UploadTaskChan chan *ImageUploadTask
	Quit           <-chan interface{} // 테스트 진행 시 Start를 끝내기 위함
}

func (uploader *DiskUploader) Start() {
	for loop := true; loop; {
		select {
		case uploadTask := <-uploader.UploadTaskChan:
			uploader.Upload(uploadTask)
		case <-uploader.Quit:
			loop = true
		}
	}
}

func (uploader *DiskUploader) Upload(task *ImageUploadTask) error {
	fmt.Println("Uploading...", task)
	fileName := task.HashedFileName
	file, err := os.Create(path.Join(task.UploadPath, fileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(task.UploadPath, 0755)
			file, err = os.Create(path.Join(task.UploadPath, fileName))
			if err != nil {
				log.Fatal(1, err)
			}
		} else {
			log.Fatal(2, err)
		}

	}
	_, ext := ParseImageFileName(fileName)
	switch ext {
	case "png":
		err := png.Encode(file, task.ImageData)
		if err != nil {
			log.Fatal(3, err)
		}
	case "jpg", "jpeg":
		err := jpeg.Encode(file, task.ImageData, nil)
		if err != nil {
			log.Fatal(4, err)
		}

	}

	return nil
}
