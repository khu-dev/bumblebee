package main

import (
    "bytes"
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify/assert"
    "io/ioutil"
    "testing"
)

// echo의 request handler에서 file을 decode 하는 과정을 테스트
func TestDecodeImageFile(t *testing.T) {
	for i, tc := range []struct{
		filename string
		expectedExt string
	}{
		{filename: "test/test_bmp.bmp", expectedExt: "png"},
		{filename: "test/test_bmp", expectedExt: "png"},
		{filename: "test/test_gif", expectedExt: "gif"},
		{filename: "test/test_gif.gif", expectedExt: "gif"},
		{filename: "test/test_jpeg", expectedExt: "jpeg"},
		{filename: "test/test_jpeg.jpg", expectedExt: "jpeg"},
		{filename: "test/test_jpeg.jpeg", expectedExt: "jpeg"},
		{filename: "test/test_png", expectedExt: "png"},
		{filename: "test/test_png.png", expectedExt: "png"},
	}{
		logrus.Infof("Test Case[%d] - %s", i, tc)
		data, err := ioutil.ReadFile(tc.filename)
		assert.NoError(t, err)
		imageData, gifImageData, ext, err := DecodeImageFile(bytes.NewReader(data))

		assert.NoError(t, err)
		assert.Equal(t, tc.expectedExt, ext)
		if tc.expectedExt == "gif"{
			assert.Nil(t, imageData)
			assert.NotNil(t, gifImageData)
		} else{
			assert.NotNil(t, imageData)
			assert.Nil(t, gifImageData)
		}
	}
}
//type parseImageFileNameTestCase struct {
//	originalFileName string
//	parsedFileName   string
//	extensionName    string
//	isError          bool
//}
//
//func TestParseImageFileName(t *testing.T) {
//	cases := []*parseImageFileNameTestCase{
//		&parseImageFileNameTestCase{
//			originalFileName: "abc.jpg",
//			parsedFileName:   "abc",
//			extensionName:    "jpeg",
//		},
//		&parseImageFileNameTestCase{
//			originalFileName: "abc.jpeg",
//			parsedFileName:   "abc",
//			extensionName:    "jpeg",
//		},
//		&parseImageFileNameTestCase{
//			originalFileName: "abc.png",
//			parsedFileName:   "abc",
//			extensionName:    "png",
//		},
//		&parseImageFileNameTestCase{
//			originalFileName: "my.name.is.jinsu.jpg",
//			parsedFileName:   "my.name.is.jinsu",
//			extensionName:    "jpeg", // jpg는 사용하지 않음.
//		},
//		&parseImageFileNameTestCase{
//			originalFileName: "my name is jinsu.jpg",
//			parsedFileName:   "my name is jinsu",
//			extensionName:    "jpeg",
//		},
//		&parseImageFileNameTestCase{
//			originalFileName: ".bit",
//			isError:          true,
//		},
//		&parseImageFileNameTestCase{
//			originalFileName: "cowboy",
//			isError:          true,
//		},
//		&parseImageFileNameTestCase{
//			originalFileName: "",
//			isError:          true,
//		},
//	}
//	for _, c := range cases {
//		t.Run(c.originalFileName, func(t *testing.T) {
//			name, ext, err := ParseImageFileName(c.originalFileName)
//			if c.isError {
//				assert.NotNil(t, err)
//			} else {
//				assert.Nil(t, err)
//				assert.Equal(t, c.parsedFileName, name)
//				assert.Equal(t, c.extensionName, ext)
//			}
//		})
//	}
//}
