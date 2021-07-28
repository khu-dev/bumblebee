package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/gif"
	"io"
	"strconv"
	"time"
)

var (
	ErrWrongImageFileName = errors.New("잘못된 형식의 이미지 파일 이름입니다. .jpg, .jpeg, .png 등의 이미지를 업로드해주세요.")
	ErrUnableToDecodeImage = errors.New(" 이미지 파일을 해석할 수 없습니다. 지원하지 않는 포맷의 이미지일 수 있습니다.")
)

// 옛날에 만들어놨는데 이제 decode를 써서 필요없을듯
//func ParseImageFileName(fileName string) (string, string, error) {
//	splitted := strings.Split(fileName, ".")
//	if len(splitted) < 2 || splitted[0] == "" {
//		return "", "", ErrWrongImageFileName
//	}
//	pureName := strings.Join(splitted[:len(splitted)-1], ".")
//	ext := splitted[len(splitted)-1]
//	switch ext {
//	case "jpeg", "jpg":
//		ext = "jpeg"
//	case "png":
//	default:
//		return "", "", ErrWrongImageFileName
//	}
//
//	return pureName, ext, nil
//}

// 현재 되는 걸로 확인된 이미지 확장자 - jpeg, jpg, png
// jpg는 jpeg로 해석됨.
// bmp는 png로 해석됨.
// gif는 로직이 많이 달라서 미지원
func DecodeImageFile(reader io.Reader) (imageData image.Image, gifImageData *gif.GIF, extension string, err error){
	imageData, extension, err = image.Decode(reader)
	if err != nil {
		if errors.Is(err, image.ErrFormat) {
			// gif는 로직이 많이 달라서 미지원
			gifImageData, err = gif.DecodeAll(reader)
			if err != nil {
				return
			}

			return
		}
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
