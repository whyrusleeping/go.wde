package sdlw

import (
	"fmt"
	"image"
	"image/color"
	"github.com/skelterjohn/go.wde"
	"github.com/whyrusleeping/sdl"
	"runtime"
	"log"
)

func init() {
	wde.BackendNewWindow = NewWindow
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		panic(err)
	}
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
	d *sdl.Display
	context sdl.GLContext
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
		switch e := e.(type) {
		case *sdl.KeyboardEvent:
			if e.Type == sdl.KEYDOWN {
				rev := new(wde.KeyDownEvent)
				rev.Key = ConvertKeyCode(e.ScanCode)
				w.keychords[rev.Key] = true
				w.events <- rev
			} else if e.Type == sdl.KEYUP {
				rev := new(wde.KeyUpEvent)
				rev.Key = ConvertKeyCode(e.ScanCode)
				w.keychords[rev.Key] = false
				w.events <- rev
			}
			chord := new(wde.KeyTypedEvent)
			chord.Chord = wde.ConstructChord(w.keychords)
			w.events <- chord
		case *sdl.MouseButtonEvent:
			fmt.Println("Mouse button event...")
			rev := new(wde.MouseButtonEvent)
			rev.Which = wde.Button(1 << e.Button)
			log.Printf("Button: %d\n",e.Button)
			rev.Where = image.Pt(e.X,e.Y)
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
				log.Printf("resize to: %d %d\n", e.Data[0], e.Data[1])
				rev := new(wde.ResizeEvent)
				rev.Height = e.Data[1]
				rev.Width = e.Data[0]
				w.events <- rev
			case sdl.WINDOWEVENT_CLOSE:
				log.Println("Close the window please.")
				w.events <- &wde.CloseEvent{}
			case sdl.WINDOWEVENT_FOCUS_GAINED:
				log.Println("Focus gained, woot!")
			case sdl.WINDOWEVENT_FOCUS_LOST:
				log.Println("Focus lost, must have ADHD.")
			case sdl.WINDOWEVENT_MOVED:
				log.Printf("please move window to %d %d.\n", e.Data[0], e.Data[1])
			default:
				log.Printf("UNRECOGNIZED WINDOW EVENT: %d\n", e.Event)
			}
		}
	}
}

func (w *Window) manageThread(width, height int, ready chan error) {
	runtime.LockOSThread()
	screen, err := sdl.NewDisplay(width, height, sdl.WINDOW_OPENGL)

	if err != nil {
		ready <- err
		return
	}

	w.context = screen.CreateGLContext()
	w.d = screen
	w.d.Present()

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
			w.d.MakeGLCurrent(w.context)
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
			w.buffer.Clear()
		}
	}
}

func ConvertKeyCode(key int32) string {
	//v, ok := keyMap[key]
	if int(key) >= len(keys) || key < 4 {
		fmt.Printf("Unrecognized keycode: %d\n",key);
		return ""
	}
	fmt.Printf("Key: %d %s\n", key, keys[key])

	return keys[key]
}
