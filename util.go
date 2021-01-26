package main

import (
    "errors"
    "strings"
)

var(
    WrongImageFileNameErr = errors.New("잘못된 형식의 이미지 파일 이름입니다. .jpg, .jpeg, .png 등의 이미지를 업로드해주세요.")
)
func ParseImageFileName(fileName string) (string, string, error) {
    splitted := strings.Split(fileName, ".")
    if len(splitted) < 2 || splitted[0] == ""{
        return "", "", WrongImageFileNameErr
    }
    pureName := strings.Join(splitted[:len(splitted)-1], ".")
    ext := splitted[len(splitted)-1]
    switch ext{
    case "jpeg", "jpg": ext = "jpeg"
    case "png":
    default:
        return "", "", WrongImageFileNameErr
    }

	return pureName, ext, nil
}
