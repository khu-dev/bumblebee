package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	WrongImageFileNameErr = errors.New("잘못된 형식의 이미지 파일 이름입니다. .jpg, .jpeg, .png 등의 이미지를 업로드해주세요.")
)

func ParseImageFileName(fileName string) (string, string, error) {
	splitted := strings.Split(fileName, ".")
	if len(splitted) < 2 || splitted[0] == "" {
		return "", "", WrongImageFileNameErr
	}
	pureName := strings.Join(splitted[:len(splitted)-1], ".")
	ext := splitted[len(splitted)-1]
	switch ext {
	case "jpeg", "jpg":
		ext = "jpeg"
	case "png":
	default:
		return "", "", WrongImageFileNameErr
	}

	return pureName, ext, nil
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
