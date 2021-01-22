package main

import (
    "strings"
)

func ParseImageFileName(fileName string) (pureName, extension string) {
	return strings.Split(fileName, ".")[0], strings.Split(fileName, ".")[1]
}
