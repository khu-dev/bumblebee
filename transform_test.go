package main

import (
	"fmt"
	"github.com/nfnt/resize"
	"github.com/stretchr/testify/assert"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	// Test시에sms Block 없이 간단하게 Test할 수 있도록 Buffered chan 이용
	transformer     *Transformer
	transformerQuit chan interface{}
)

func BeforeEachTransformTest(tb testing.TB) {
	transformerQuit = make(chan interface{}, 100)
	transformer = &Transformer{
		ResizeTaskChan:    make(chan *ImageResizeTask, 100),
		ThumbnailTaskChan: make(chan *ImageGenerateThumbnailTask, 100),
		UploadTaskChan:    make(chan *ImageUploadTask, 100),
		Quit:              transformerQuit,
	}
}

func AfterEachTransformTest(tb testing.TB) {
	transformerQuit = nil
	transformer = nil
}

// test나 benchmark시에 사용하는 sample image를 다운받는다.
func downloadSampleImage(tb testing.TB) image.Image{
	resp, err := http.Get("https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png")
	assert.NoError(tb, err)
	assert.NotNil(tb, resp)
	defer resp.Body.Close()
	imageData, err := png.Decode(resp.Body)
	assert.NoError(tb, err)
	assert.NotNil(tb, imageData)
	return imageData
}

// downloadSampleImage를 testing.T.Run으로 감싸기 위함.
func DownloadSampleImage(t *testing.T) image.Image {
	var imageData image.Image
	t.Run("구글_로고_로드", func(t *testing.T) {
		imageData = downloadSampleImage(t)
	})

	return imageData
}

func TestTransformer_Resize(t *testing.T) {
	BeforeEachTransformTest(t)
	defer AfterEachTransformTest(t)
	var imageData image.Image = DownloadSampleImage(t)
	assert.NotNil(t, imageData)
	imageResizeTask := &ImageResizeTask{
		BaseImageTask: &BaseImageTask{
			OriginalFileName: "google_logo.png",
			ImageData:        imageData,
		}, MaxWidth: 128, MaxHeight: 128,
	}
	transformer.Resize(imageResizeTask)
	assert.Equal(t, 0, imageResizeTask.ResizedImageData.Bounds().Min.X)
	assert.Equal(t, 0, imageResizeTask.ResizedImageData.Bounds().Min.Y)
	assert.Equal(t, 128, imageResizeTask.ResizedImageData.Bounds().Max.X)
	assert.Equal(t, 128, imageResizeTask.ResizedImageData.Bounds().Max.Y)
}

func TestTransformer_GenerateThumbnail(t *testing.T) {
	BeforeEachTransformTest(t)
	defer AfterEachTransformTest(t)

	var imageData image.Image = DownloadSampleImage(t)
	assert.NotNil(t, imageData)
	imageThumbnailTask := &ImageGenerateThumbnailTask{
		BaseImageTask: &BaseImageTask{
			OriginalFileName: "google_logo.png",
			ImageData:        imageData,
		},
	}
	transformer.GenerateThumbnail(imageThumbnailTask)
	assert.Equal(t, 0, imageThumbnailTask.ThumbnailImageData.Bounds().Min.X)
	assert.Equal(t, 0, imageThumbnailTask.ThumbnailImageData.Bounds().Min.Y)
	assert.Equal(t, ThumbnailWidth, imageThumbnailTask.ThumbnailImageData.Bounds().Max.X)
	assert.Equal(t, ThumbnailHeight, imageThumbnailTask.ThumbnailImageData.Bounds().Max.Y)
}

func TestTransformer_Start(t *testing.T) {
	t.Run("썸네일", func(t *testing.T) {
		BeforeEachTransformTest(t)
		defer AfterEachTransformTest(t)
		go transformer.Start()
		imageData := DownloadSampleImage(t)
		transformer.ThumbnailTaskChan <- &ImageGenerateThumbnailTask{
			BaseImageTask: &BaseImageTask{
				OriginalFileName: "google_logo.png",
				ImageData:        imageData,
			},
		}
		select {
		case <-transformer.UploadTaskChan:
		case <-time.After(10 * time.Second):
			t.Fatal("[TimeOutError] Thumbnail 생성 후 UploadTaskChan에 Message가 들어오지 않습니다.")
		}
		transformerQuit <- struct{}{}
	})

	t.Run("리사이즈", func(t *testing.T) {
		BeforeEachTransformTest(t)
		defer AfterEachTransformTest(t)
		go transformer.Start()
		imageData := DownloadSampleImage(t)
		transformer.ResizeTaskChan <- &ImageResizeTask{
			BaseImageTask: &BaseImageTask{
				OriginalFileName: "google_logo.png",
				ImageData:        imageData,
			}, MaxWidth: 128, MaxHeight: 128,
		}
		select {
		case <-transformer.UploadTaskChan:
		case <-time.After(5 * time.Second):
			t.Fatal("[TimeOutError] Resize 후 UploadTaskChan에 Message가 들어오지 않습니다.")
		}
		transformerQuit <- struct{}{}
	})
}

// concurrent benchmark를 위한 것
func (t *Transformer) resizeBenchmarkConcurrent(task *ImageResizeTask) {
	task.ResizedImageData = resize.Resize(uint(task.MaxWidth), uint(task.MaxHeight), task.ImageData, resize.Lanczos3)
	task.ImageData = nil // 이제 필요 없으니 지워줘서 GC가 처리할 수 있게 함.
	t.UploadTaskChan <- &ImageUploadTask{
		BaseImageTask: &BaseImageTask{
			OriginalFileName: task.OriginalFileName,
			HashedFileName:   task.HashedFileName,
			ImageData:        task.ResizedImageData,
		},
	}
}

func BenchmarkTransformer_Start(b *testing.B) {
	//b.Run("리사이즈에", func(b *testing.B) {

	file, err := os.Open("test_data_wallpaper.jpg")
	if err != nil {
		log.Fatal(err)
	}

	// decode jpeg into image.Image
	imageData, err := jpeg.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()
	numOfTask := 50

	for sizeOfWorkerPool := 1; sizeOfWorkerPool < 11; sizeOfWorkerPool += 2{
		b.Run(fmt.Sprintf("%d_task_worker_pool_%d", numOfTask, sizeOfWorkerPool), func(b *testing.B) {
		BeforeEachTransformTest(b)
		defer AfterEachTransformTest(b)
		go transformer.Start()
		go transformer.Start()
		go transformer.Start()
		go transformer.Start()
		go transformer.Start()


		go func() {
			for i := 0; i < numOfTask; i++{
				transformer.ResizeTaskChan <- &ImageResizeTask{
					BaseImageTask: &BaseImageTask{
						OriginalFileName: "google_logo.png",
						ImageData:        imageData,
					}, MaxWidth: 128, MaxHeight: 128,
				}
			}
		}()
		finishedCNT := 0
		for ; finishedCNT <numOfTask; finishedCNT++{
			output := <-transformer.UploadTaskChan
			assert.NotNil(b, output.ImageData)
		}
		transformerQuit <- struct{}{}
	})

	}

	b.Run(fmt.Sprintf("%d_task_unlimited_concurrency", numOfTask), func(b *testing.B) {
		BeforeEachTransformTest(b)
		defer AfterEachTransformTest(b)
		go func() {
			for i := 0; i < numOfTask; i++{
				transformer.ResizeTaskChan <- &ImageResizeTask{
					BaseImageTask: &BaseImageTask{
						OriginalFileName: "google_logo.png",
						ImageData:        imageData,
					}, MaxWidth: 128, MaxHeight: 128,
				}
			}
		}()

		for rep := 0; rep < numOfTask; rep++{
			task := <-transformer.ResizeTaskChan
			go transformer.resizeBenchmarkConcurrent(task)
		}

		finishedCNT := 0
		for finishedCNT = 0; finishedCNT < numOfTask; finishedCNT++{
			<-transformer.UploadTaskChan
		}
		transformerQuit <- struct{}{}
	})
}