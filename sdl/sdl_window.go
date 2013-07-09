package sdlw

import (
	"github.com/skelterjohn/go.wde"
	"github.com/whyrusleeping/sdl"
	"runtime"
)

func init() {
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		panic(err)
	}
}

type Window struct {
	w *sdl.Window
	lock bool

	closed bool

	events chan interface{}

	chTitle chan string
	chSize chan point
}

type point struct {
	x,y int
}

func NewWindow(width, height int)  (w *Window, err error) {
	w = new(Window)

	w.events = make(chan interface{})
	w.chSize = make(chan point)
	w.chTitle = make(chan string)

	ready := make(chan error)
	go w.manageThread(ready)
	err = <-ready
	return
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
	width, height = w.w.Size()
	return
}

func (w *Window) LockSize(lock bool) {
	w.lock = lock
}

func (w *Window) EventChan() <-chan interface{} {
	return w.events
}

func (w *Window) Close() error {
	w.w.Destroy()
	w.w.Free()
	close(w.events)
	close(w.chSize)
	close(w.chTitle)
	w.closed = true
	return nil
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

func (w *Window) manageThread(ready chan error) {
	runtime.LockOSThread()
	screen, err := sdl.NewDisplay(width, height, sdl.WINDOW_OPENGL)
	if err != nil {
		ready <- err
		return
	}
	w.w = screen

	go w.collectEvents()

	ready <- nil
	for {
		select {
		case s := <-w.chSize:
			if !w.lock {
				w.w.SetSize(width, height)
			}
		case title := <-w.chTitle:
			w.SetTitle(title)

		}
	}
}
