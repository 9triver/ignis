package main

import (
	"bytes"
	"image"
	"log"

	"github.com/disintegration/imaging"
)

const Name = "ResizeImage"

type Input struct {
	Configs map[string]any
	Image   image.Image
}

type Output = []byte

func mapGet[T any](m map[string]any, key string, orElse T) T {
	v, ok := m[key]
	if !ok {
		return orElse
	}

	t, ok := v.(T)
	if !ok {
		return orElse
	}

	return t
}

func Impl(input Input) (Output, error) {
	width := mapGet(input.Configs, "width", 1280)
	height := mapGet(input.Configs, "height", 1280)

	resized := imaging.Resize(input.Image, width, height, imaging.BSpline)

	log.Printf("%s: %dx%d", Name, width, height)

	buf := bytes.NewBuffer(nil)
	err := imaging.Encode(buf, resized, imaging.JPEG)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
