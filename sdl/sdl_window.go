package sdlw

import (
	"fmt"
	"image"
	"github.com/skelterjohn/go.wde"
	"github.com/jackyb/go-sdl2/sdl"
	"runtime"
	"log"
)

func init() {
	wde.BackendNewWindow = NewWindow
	sdl.Init(sdl.INIT_EVERYTHING)

	ch := make(chan struct{}, 1)
	wde.BackendRun = func() {
		<-ch
	}
	wde.BackendStop = func() {
		ch <- struct{}{}
		sdl.Quit()
	}
}

type Window struct {
	w *sdl.Window
	r *sdl.Renderer
	buffer *SdlBuffer
	lock bool

	closed bool

	events chan interface{}

	chTitle chan string
	chSize chan point
	chFlush chan struct{}
	chShow chan struct{}

	width, height int
	keychords map[string]bool
}

type point image.Point

func NewWindow(width, height int)  (wde.Window, error) {
	fmt.Println("new window!!")
	w := new(Window)
	w.width = width
	w.height = height

	w.events = make(chan interface{})
	w.chSize = make(chan point)
	w.chTitle = make(chan string)
	w.chFlush = make(chan struct{})
	w.chShow = make(chan struct{})

	w.buffer = NewSdlBuffer(width, height)
	w.keychords	= make(map[string]bool)

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
	return w.w.GetSize()
}

func (w *Window) LockSize(lock bool) {
	w.lock = lock
}

func (w *Window) EventChan() <-chan interface{} {
	return w.events
}

func (w *Window) Close() error {
	w.w.Destroy()
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
		switch e := e.(type) {
		case *sdl.KeyDownEvent:
			rev := new(wde.KeyDownEvent)
			rev.Key = ConvertKeyCode(e.Keysym.Scancode)
			w.keychords[rev.Key] = true
			w.events <- rev
			chord := new(wde.KeyTypedEvent)
			chord.Chord = wde.ConstructChord(w.keychords)
			w.events <- chord
		case *sdl.KeyUpEvent:
			rev := new(wde.KeyUpEvent)
			rev.Key = ConvertKeyCode(e.Keysym.Scancode)
			w.keychords[rev.Key] = false
			w.events <- rev
			chord := new(wde.KeyTypedEvent)
			chord.Chord = wde.ConstructChord(w.keychords)
			w.events <- chord
		case *sdl.MouseButtonEvent:
			fmt.Println("Mouse button event...")
			rev := new(wde.MouseButtonEvent)
			rev.Which = wde.Button(1 << e.Button)
			log.Printf("Button: %d\n",e.Button)
			rev.Where = image.Pt(int(e.X),int(e.Y))
			w.events <- rev
		case *sdl.MouseMotionEvent:

		case *sdl.MouseWheelEvent:
		case *sdl.QuitEvent:
			w.events <- new(wde.CloseEvent)
		case *sdl.WindowEvent:
			switch e.Event {
				//http://wiki.libsdl.org/moin.fcg/SDL_WindowEvent
			case sdl.WINDOWEVENT_SHOWN:
				log.Println("Window shown!")
			case sdl.WINDOWEVENT_RESTORED:
				log.Println("Window restored.")
			case sdl.WINDOWEVENT_EXPOSED:
				log.Println("Window exposed, whatever that means...")
			case sdl.WINDOWEVENT_HIDDEN:
				log.Println("Window hidden.. sneaky thing.")
			case sdl.WINDOWEVENT_MAXIMIZED:
				log.Println("Window Maximized!")
			case sdl.WINDOWEVENT_MINIMIZED:
				log.Println("Window Minimized!")
			case sdl.WINDOWEVENT_ENTER:
				w.events <- new(wde.MouseEnteredEvent)
				log.Println("Mouse enter...")
			case sdl.WINDOWEVENT_LEAVE:
				w.events <- new(wde.MouseExitedEvent)
				log.Println("Mouse leave...")
			case sdl.WINDOWEVENT_RESIZED:
				log.Printf("resize to: %d %d\n", e.Data1, e.Data2)
				rev := new(wde.ResizeEvent)
				rev.Height = int(e.Data1)
				rev.Width = int(e.Data2)
				w.events <- rev
			case sdl.WINDOWEVENT_CLOSE:
				log.Println("Close the window please.")
				w.events <- &wde.CloseEvent{}
			case sdl.WINDOWEVENT_FOCUS_GAINED:
				log.Println("Focus gained, woot!")
			case sdl.WINDOWEVENT_FOCUS_LOST:
				log.Println("Focus lost, must have ADHD.")
			case sdl.WINDOWEVENT_MOVED:
				log.Printf("please move window to %d %d.\n", e.Data1, e.Data2)
			default:
				log.Printf("UNRECOGNIZED WINDOW EVENT: %d\n", e.Event)
			}
		}
	}
}

func (w *Window) manageThread(width, height int, ready chan error) {
	runtime.LockOSThread()
	window, render := sdl.CreateWindowAndRenderer(width, height, sdl.WINDOW_OPENGL)

	w.w = window
	w.r = render

	w.w.Show()

	go w.collectEvents()

	ready <- nil
	for {
		select {
		case s := <-w.chSize:
			if !w.lock {
				w.w.SetSize(s.X, s.Y)
			}
		case title := <-w.chTitle:
			w.w.SetTitle(title)
		case <-w.chShow:
			w.w.Show()
		case <-w.chFlush:
			for x := 0; x < w.width; x++ {
				for y := 0; y < w.height; y++ {
					r,g,b,a := w.buffer.At(x,y).RGBA()
					w.r.SetDrawColor(uint8(r),uint8(g),uint8(b),uint8(a))
					w.r.DrawPoint(x,y)
				}
			}
			w.r.Present()
			w.buffer.Clear()
		}
	}
}

func ConvertKeyCode(key sdl.Scancode) string {
	//v, ok := keyMap[key]
	if int(key) >= len(keys) || key < 4 {
		fmt.Printf("Unrecognized keycode: %d\n",key);
		return ""
	}
	ikey := int(key)
	fmt.Printf("Key: %d %s\n", key, keys[ikey])

	return keys[ikey]
}
