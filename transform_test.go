package main

import (
    "github.com/stretchr/testify/assert"
    "image"
    "image/png"
    "net/http"
    "testing"
    "time"
)

var(
    // Test시에sms Block 없이 간단하게 Test할 수 있도록 Buffered chan 이용
    transformer *Transformer
    transformerQuit chan interface{}
)

func BeforeEachTransformTest(){
    transformerQuit = make(chan interface{}, 100)
    transformer = &Transformer{
        ResizeTaskChan: make(chan *ImageResizeTask, 100),
        ThumbnailTaskChan: make(chan *ImageGenerateThumbnailTask, 100),
        UploadTaskChan: make(chan *ImageUploadTask, 100),
        Quit: transformerQuit,
    }
}

func AfterEachTransformTest(){
    transformerQuit = nil
    transformer = nil
}
func DownloadSampleImage(t *testing.T) image.Image{
    var imageData image.Image
    t.Run("구글에서_로고이미지_데이터_가져오기", func(t *testing.T) {
        resp, err := http.Get("https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png")
        assert.NoError(t, err)
        assert.NotNil(t, resp)
        defer resp.Body.Close()
        imageData, err = png.Decode(resp.Body)
        assert.NoError(t, err)
        assert.NotNil(t, imageData)
    })

    return imageData
}

func TestTransformer_Resize(t *testing.T) {
    BeforeEachTransformTest()
    defer AfterEachTransformTest()
    var imageData image.Image = DownloadSampleImage(t)
    assert.NotNil(t, imageData)
    imageResizeTask := &ImageResizeTask{
        BaseImageTask: &BaseImageTask{
            OriginalFileName: "google_logo.png",
            ImageData: imageData,
        }, MaxWidth: 128, MaxHeight: 128,
    }
    transformer.Resize(imageResizeTask)
    assert.Equal(t, 0, imageResizeTask.ResizedImageData.Bounds().Min.X)
    assert.Equal(t, 0, imageResizeTask.ResizedImageData.Bounds().Min.Y)
    assert.Equal(t, 128, imageResizeTask.ResizedImageData.Bounds().Max.X)
    assert.Equal(t, 128, imageResizeTask.ResizedImageData.Bounds().Max.Y)
}

func TestTransformer_GenerateThumbnail(t *testing.T) {
    BeforeEachTransformTest()
    defer AfterEachTransformTest()

    var imageData image.Image = DownloadSampleImage(t)
    assert.NotNil(t, imageData)
    imageThumbnailTask := &ImageGenerateThumbnailTask{
        BaseImageTask: &BaseImageTask{
            OriginalFileName: "google_logo.png",
            ImageData: imageData,
        },
    }
    transformer.GenerateThumbnail(imageThumbnailTask)
    assert.Equal(t, 0, imageThumbnailTask.ThumbnailImageData.Bounds().Min.X)
    assert.Equal(t, 0, imageThumbnailTask.ThumbnailImageData.Bounds().Min.Y)
    assert.Equal(t, ThumbnailWidth, imageThumbnailTask.ThumbnailImageData.Bounds().Max.X)
    assert.Equal(t, ThumbnailHeight, imageThumbnailTask.ThumbnailImageData.Bounds().Max.Y)
}

func TestTransformer_Start(t *testing.T) {
    t.Run("썸네일에_대한_작업을_잘_수행하는가", func(t *testing.T) {
        BeforeEachTransformTest()
        defer AfterEachTransformTest()
        go transformer.Start()
        imageData := DownloadSampleImage(t)
        transformer.ThumbnailTaskChan <- &ImageGenerateThumbnailTask{
            BaseImageTask: &BaseImageTask{
                OriginalFileName: "google_logo.png",
                ImageData: imageData,
            },
        }
        select{
        case <- transformer.UploadTaskChan:
        case <- time.After(10 * time.Second):
            t.Fatal("[TimeOutError] Thumbnail 생성 후 UploadTaskChan에 Message가 들어오지 않습니다.")
        }
        transformerQuit <- struct{}{}
    })

    t.Run("리사이즈에_대한_작업을_잘_수행하는가", func(t *testing.T) {
        BeforeEachTransformTest()
        defer AfterEachTransformTest()
        go transformer.Start()
        imageData := DownloadSampleImage(t)
        transformer.ResizeTaskChan <- &ImageResizeTask{
            BaseImageTask: &BaseImageTask{
                OriginalFileName: "google_logo.png",
                ImageData: imageData,
            }, MaxWidth: 128, MaxHeight: 128,
        }
        select{
        case <- transformer.UploadTaskChan:
        case <- time.After(5 * time.Second):
            t.Fatal("[TimeOutError] Resize 후 UploadTaskChan에 Message가 들어오지 않습니다.")
        }
        transformerQuit <- struct{}{}
    })
}