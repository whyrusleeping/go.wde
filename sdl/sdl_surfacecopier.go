package sdlw

import (
	"image"
	"image/color"
)

type SdlBuffer struct {
	*image.RGBA
}

func NewSdlBuffer(width, height int) *SdlBuffer {
	s := new(SdlBuffer)
	s.RGBA = image.NewRGBA(image.Rect(0,0,width,height))
	return s
}

func (s *SdlBuffer) CopyRGBA(src *image.RGBA, r image.Rectangle) {
	xbound := r.Size().X
	ybound := r.Size().Y
	xst := r.Min.X
	yst := r.Min.Y
	for x := 0; x < xbound; x++ {
		for y := 0; y < ybound; y++ {
			s.Set(xst + x, yst + y, src.At(x,y))
		}
	}
}

func (s *SdlBuffer) Clear() {
	c := color.RGBA{0,0,0,0}
	r := s.Bounds().Size()
	for x := 0; x < r.X; x++ {
		for y := 0; y < r.Y; y++ {
			s.Set(x,y,c)
		}
	}
}
