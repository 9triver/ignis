package camera

import (
	"time"

	"github.com/blackjack/webcam"
)

type Camera struct {
	webcam *webcam.Webcam
	device string
}

func (c *Camera) Reset(device string) error {
	c.Close()

	c.device = device
	cam, err := webcam.Open(c.device)
	if err != nil {
		return err
	}

	if err := cam.StartStreaming(); err != nil {
		return err
	}

	c.webcam = cam
	return nil
}

func (c *Camera) Close() error {
	if c.webcam == nil {
		return nil
	}

	if err := c.webcam.StopStreaming(); err != nil {
		return err
	}
	if err := c.webcam.Close(); err != nil {
		return err
	}
	c.webcam = nil

	return nil
}

func (c *Camera) Capture(timeout time.Duration) ([]byte, error) {
	if err := c.webcam.WaitForFrame(uint32(timeout) / 1000); err != nil {
		return nil, err
	}
	return c.webcam.ReadFrame()
}

func New() *Camera {
	return &Camera{}
}
