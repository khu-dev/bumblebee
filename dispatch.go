package main

import "github.com/sirupsen/logrus"

func DispatchMessages(baseImageTask *BaseImageTask){
    // base image task를 복제하면 imageData가 복제되어 메모리를 너무 많이 점유하지는 않을까?
	// => imageData안에는 결국 byte arr의 데이터가 들어있을텐데, 이는 = 할당을 해도 deepcopy 되는 것이아니라
	// 같은 arr을 참조하는 slice일 뿐임.

	// Enqueue 섬네일 생성 작업
	go func() {
	    ThumbnailTaskChan <- &ImageGenerateThumbnailTask{
	        BaseImageTask: baseImageTask,
        }
        logrus.Print("Enqueued thumbnail task")
	}()

	// Enqueue 리사이징 생성 작업
	go func(){
	    for _, size := range ResizeSizes{
	        ResizeTaskChan <- &ImageResizeTask{
	            BaseImageTask: baseImageTask,
	            ResizingWidth: size,
            }
            logrus.Print("Enqueued resize task")
        }
    }()

	// Upload original image
	go func(){
	    UploadTaskChan <- &ImageUploadTask{
	        BaseImageTask: baseImageTask,
	        UploadPath: "original",
        }
        logrus.Print("Enqueued upload task")
    }()
}
