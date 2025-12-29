package main

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
)

const Name = "CollectImage"

type Input struct {
	Configs map[string]any
}

type Output image.Image

func Impl(input Input) (Output, error) {
	pathV, ok := input.Configs["path"]
	if !ok {
		return nil, errors.New("missing input path")
	}

	path, ok := pathV.(string)
	if !ok {
		return nil, errors.New("incorrect type for path, must be string")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	im, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	log.Printf("%s: read image %dx%d, format %s\n", Name, im.Bounds().Dx(), im.Bounds().Dy(), format)

	return im, nil
}
