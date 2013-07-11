package sdlw

import (
	"image"
	"image/color"
	"github.com/skelterjohn/go.wde"
	"github.com/whyrusleeping/sdl"
	"runtime"
)

func init() {
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		panic(err)
	}
	wde.BackendNewWindow = NewWindow
	ch := make(chan struct{}, 1)
	wde.BackendRun = func() {
		<-ch
	}
	wde.BackendStop = func() {
		ch <- struct{}{}
	}
}

type Window struct {
	d *sdl.Display
	buffer *SdlBuffer
	lock bool

	closed bool

	events chan interface{}

	chTitle chan string
	chSize chan point
	chFlush chan struct{}
	chShow chan struct{}

	width, height int
}

type point image.Point

func NewWindow(width, height int)  (wde.Window, error) {
	w := new(Window)

	w.events = make(chan interface{})
	w.chSize = make(chan point)
	w.chTitle = make(chan string)
	w.chFlush = make(chan struct{})
	w.chShow = make(chan struct{})

	w.buffer = NewSdlBuffer(width, height)

	ready := make(chan error)
	go w.manageThread(width, height, ready)
	err := <-ready
	return w, err
}

///////////////////
//Interface methods
///////////////////

func (w *Window) SetTitle(title string) {
	if w.closed {
		return
	}
	w.chTitle <- title
}

func (w *Window) SetSize(width, height int) {
	if w.closed {
		return
	}
	w.chSize <- point{width,height}
}

func (w *Window) Size() (width, height int) {
	if w.closed {
		return
	}
	width, height = w.d.Size()
	return
}

func (w *Window) LockSize(lock bool) {
	w.lock = lock
}

func (w *Window) EventChan() <-chan interface{} {
	return w.events
}

func (w *Window) Close() error {
	w.d.Window.Destroy()
	close(w.events)
	close(w.chSize)
	close(w.chTitle)
	w.closed = true
	return nil
}

func (w *Window) FlushImage(r ...image.Rectangle) {
	if w.closed {
		return
	}
	w.chFlush <- struct{}{}
}

func (w *Window) Screen() wde.Image {
	return w.buffer
}

func (w *Window) Show() {
	w.chShow <- struct{}{}
}

///////////////////////
//Non interface methods
///////////////////////
func (w *Window) collectEvents() {
	for {
		e := sdl.PollEvent()
		if e == nil {
			continue
		}
		//Event translation
	}
}

func (w *Window) manageThread(width, height int, ready chan error) {
	runtime.LockOSThread()
	screen, err := sdl.NewDisplay(width, height, sdl.WINDOW_OPENGL)
	if err != nil {
		ready <- err
		return
	}
	w.d = screen

	go w.collectEvents()

	ready <- nil
	for {
		select {
		case s := <-w.chSize:
			if !w.lock {
				w.d.SetSize(s.X, s.Y)
			}
		case title := <-w.chTitle:
			w.d.SetTitle(title)
		case <-w.chShow:
			w.d.Show()
		case <-w.chFlush:
			c := color.RGBA{}
			for x := 0; x < w.width; x++ {
				for y := 0; y < w.height; y++ {
					r,g,b,a := w.buffer.At(x,y).RGBA()
					c.R = uint8(r)
					c.G = uint8(g)
					c.B = uint8(b)
					c.A = uint8(a)
					w.d.SetDrawColor(c)
					w.d.DrawPoint(x,y)
				}
			}
			w.d.Present()
		}
	}
}
