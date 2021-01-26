package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"image"
	"image/jpeg"
	"image/png"
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
	parsedName, ext, err := ParseImageFileName(inputFileName)
	if err != nil{
	    return c.JSON(400, map[string]interface{}{
	        "data": nil,
	        "message": WrongImageFileNameErr.Error(),
        })
    }
    hashedFileName := getHashedFileName(parsedName) + "." + ext
	src, err := input.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	var imageData image.Image
	switch ext{
    case "jpg", "jpeg":
        imageData, err = jpeg.Decode(src)
    case "png":
        imageData, err = png.Decode(src)
    default:
        return c.JSON(400, BaseResponse{
            Message: WrongImageFileNameErr.Error(),
        })
    }

	DispatchMessages(&BaseImageTask{
		ImageData: imageData,
		OriginalFileName: inputFileName,
		HashedFileName: hashedFileName,
	})

	fmt.Println(viper.GetString("storage.aws.endpoint"))
	return c.JSON(200, GenerateSuccessfullyUploadedResponse(hashedFileName))
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

func GenerateSuccessfullyUploadedResponse (hashedFileName string)*BaseResponse{
	rootEndpoint := viper.GetString("storage.aws.endpoint")
	return &BaseResponse{
		Data: SuccessfullyUploadedResponseData{
			RootEndpoint: rootEndpoint,
			FileName: hashedFileName,
			ThumbnailURL: path.Join(rootEndpoint, "thumbnail", hashedFileName),
			Resized256URL: path.Join(rootEndpoint, "resized", strconv.Itoa(256), hashedFileName),
			Resized1024URL: path.Join(rootEndpoint, "resized", strconv.Itoa(1024), hashedFileName),
		},
	}

}

func ForceContentTypeMultipartFormDataMiddleware(handlerFunc echo.HandlerFunc) echo.HandlerFunc {
	return func(context echo.Context) error {
		logrus.Warn(context.Request().Header.Get("Content-Type"))
		if !strings.HasPrefix(context.Request().Header.Get("Content-Type"), "multipart/form-data"){
			resp := BaseResponse{Message: "Unsupported Content-Type. Please use multipart/form-data."}
			logrus.Error(resp)
			return context.JSON(400, resp)
		}
		return handlerFunc(context)
	}
}
