package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"image"
	"path"
	"strconv"
	"strings"
)

func NewEcho() *echo.Echo{
    e := echo.New()
    e.Pre(middleware.RemoveTrailingSlash())

    g := e.Group("api")
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${method} ${status} uri=${uri} latency=${latency}\n",
		Skipper: func(context echo.Context) bool {
	  		// health check log는 너무 verbose함.
			if context.Request().URL.RequestURI() == "/healthz"{
				return true
			}
			return false
	  },
	}))
	e.GET("/healthz", func(c echo.Context) error { return c.String(200, "OK") })
    g.POST("/images", ImageUploadRequestHandler, ForceContentTypeMultipartFormDataMiddleware)

    return e
}

func ImageUploadRequestHandler(c echo.Context) error{
    // Source
	input, err := c.FormFile("image")
	if err != nil {
	    logrus.Error(err)
		return err
	}
	inputFileName := input.Filename
	if err != nil{
		logrus.Error(err)
	    return c.JSON(400, map[string]interface{}{
	        "data": nil,
	        "message": WrongImageFileNameErr.Error(),
        })
    }
    var hashedFileName string
    if c.FormValue("hashing") == "false"{
		splited := strings.Split(inputFileName, ".")
		hashedFileName = strings.Join(splited[:len(splited)-1], ".")
		logrus.Println("Omit hashing. not hashed name:", hashedFileName)
	} else{
    	hashedFileName = getHashedFileName(inputFileName)
    	logrus.Println("Hashed", inputFileName, "into", hashedFileName)
	}
	src, err := input.Open()
	if err != nil {
		logrus.Error(err)
		return err
	}
	defer src.Close()
	imageData, ext, err := image.Decode(src)

	DispatchMessages(&BaseImageTask{
		ImageData: imageData,
		OriginalFileName: inputFileName,
		HashedFileName: hashedFileName,
		Extension: ext,
	})

	resp := GenerateSuccessfullyUploadedResponse(hashedFileName + "." + ext)
	logrus.Println(resp)
	return c.JSON(200, resp)
}

type BaseResponse struct{
    Data interface{} `json:"data"`
    Message string `json:"message"`
}

type SuccessfullyUploadedResponseData struct{
	RootEndpoint string `json:"root_endpoint"`
	FileName string `json:"file_name"`
	ThumbnailURL string `json:"thumbnail_url"`
	Resized256URL string `json:"resized_256_url"`
	Resized1024URL string `json:"resized_1024_url"`
}

func GenerateSuccessfullyUploadedResponse (fileFullName string)*BaseResponse{
	rootEndpoint := viper.GetString("storage.aws.endpoint")
	return &BaseResponse{
		Data: SuccessfullyUploadedResponseData{
			RootEndpoint: rootEndpoint,
			FileName: fileFullName,
			ThumbnailURL: path.Join(rootEndpoint, "thumbnail", fileFullName),
			Resized256URL: path.Join(rootEndpoint, "resized", strconv.Itoa(256), fileFullName),
			Resized1024URL: path.Join(rootEndpoint, "resized", strconv.Itoa(1024), fileFullName),
		},
	}

}

func ForceContentTypeMultipartFormDataMiddleware(handlerFunc echo.HandlerFunc) echo.HandlerFunc {
	return func(context echo.Context) error {

		if !strings.HasPrefix(context.Request().Header.Get("Content-Type"), "multipart/form-data"){
			logrus.Warn("Content-Type in Request", context.Request().Header)
			resp := BaseResponse{Message: "Unsupported Content-Type " + context.Request().Header.Get("Content-Type") + ". Please use multipart/form-data."}
			logrus.Error(resp)
			return context.JSON(400, resp)
		}
		return handlerFunc(context)
	}
}
