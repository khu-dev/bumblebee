package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type parseImageFileNameTestCase struct {
	originalFileName string
	parsedFileName   string
	extensionName    string
	isError          bool
}

func TestParseImageFileName(t *testing.T) {
	cases := []*parseImageFileNameTestCase{
		&parseImageFileNameTestCase{
			originalFileName: "abc.jpg",
			parsedFileName:   "abc",
			extensionName:    "jpeg",
		},
		&parseImageFileNameTestCase{
			originalFileName: "abc.jpeg",
			parsedFileName:   "abc",
			extensionName:    "jpeg",
		},
		&parseImageFileNameTestCase{
			originalFileName: "abc.png",
			parsedFileName:   "abc",
			extensionName:    "png",
		},
		&parseImageFileNameTestCase{
			originalFileName: "my.name.is.jinsu.jpg",
			parsedFileName:   "my.name.is.jinsu",
			extensionName:    "jpeg", // jpg는 사용하지 않음.
		},
		&parseImageFileNameTestCase{
			originalFileName: "my name is jinsu.jpg",
			parsedFileName:   "my name is jinsu",
			extensionName:    "jpeg",
		},
		&parseImageFileNameTestCase{
			originalFileName: ".bit",
			isError:          true,
		},
		&parseImageFileNameTestCase{
			originalFileName: "cowboy",
			isError:          true,
		},
		&parseImageFileNameTestCase{
			originalFileName: "",
			isError:          true,
		},
	}
	for _, c := range cases {
		t.Run(c.originalFileName, func(t *testing.T) {
			name, ext, err := ParseImageFileName(c.originalFileName)
			if c.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, c.parsedFileName, name)
				assert.Equal(t, c.extensionName, ext)
			}
		})
	}
}
