package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/sirupsen/logrus"
	"image"
	"image/gif"
	"io"
	"io/ioutil"
	"strconv"
	"time"
)

var (
	ErrWrongImageFileName  = errors.New("잘못된 형식의 이미지 파일 이름입니다. .jpg, .jpeg, .png 등의 이미지를 업로드해주세요.")
	ErrUnableToDecodeImage = errors.New(" 이미지 파일을 해석할 수 없습니다. 지원하지 않는 포맷의 이미지일 수 있습니다.")
)

// 현재 되는 걸로 확인된 이미지 확장자 - jpeg, jpg, png
// jpg는 jpeg로 해석됨.
// bmp는 png로 해석됨.
// gif는 로직이 많이 달라서 미지원
func DecodeImageFile(reader io.Reader) (imageData image.Image, gifImageData *gif.GIF, extension string, err error) {
	// reader는 한 번만 읽을 수 있으므로 복사해둔다.
	tmpData, err := ioutil.ReadAll(reader)
	if err != nil {
		logrus.Error(err)
		return
	}

	imageData, extension, err = image.Decode(bytes.NewReader(tmpData))

	// gif package의 init에 extension 등록이 있음.
	// 따라서 gif package를 import하지 않으면 gif도 ErrFormat처리됨
	if extension == "gif" {
		imageData = nil
		gifImageData, err = gif.DecodeAll(bytes.NewReader(tmpData))
		if err != nil {
			logrus.Error(err)
			return
		}
	}
	if err != nil {
		logrus.Error(err)
		return
	}

	return
}

// extension이 포함되어있든 아니든 어차피 해싱할것이라 상관없음.
// fileName과 시간을 이용해 해싱.
func getHashedFileName(fileName string) string {
	hash := sha256.New()
	unixTimeStr := strconv.Itoa(int(time.Now().Unix()))
	hash.Write([]byte(fileName + unixTimeStr))
	md := hash.Sum(nil)
	return hex.EncodeToString([]byte(md))
}
