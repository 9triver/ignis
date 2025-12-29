package main

import (
	"image"
	"log"

	"github.com/disintegration/imaging"
)

const Name = "CentralCrop"

type Input struct {
	Image image.Image
}

type Output image.Image

func Impl(input Input) (Output, error) {
	im := input.Image

	width := min(im.Bounds().Dx(), im.Bounds().Dy())
	squared := imaging.Fill(im, width, width, imaging.Center, imaging.Lanczos)

	log.Printf("%s: %dx%d", Name, width, width)

	return squared, nil
}
