package sdlw

import (
	"fmt"
	"image"
	"github.com/skelterjohn/go.wde"
	"github.com/jackyb/go-sdl2/sdl"
	"runtime"
	"log"
)

var windowList []*Window
var newWindow chan *Window
var windowShow chan *Window
var windowFlush chan *Window
var windowChSize chan *Window
var windowTitle chan *Window
var active *Window
var keychords map[string]bool

var events chan interface{}

func init() {
	fmt.Println("Initializing!")
	wde.BackendNewWindow = NewWindow
	e := sdl.Init(sdl.INIT_EVERYTHING)
	fmt.Printf("SDL_Init returned: %d\n", e)

	keychords	= make(map[string]bool)

	newWindow = make(chan *Window)
	windowShow = make(chan *Window)
	windowFlush = make(chan *Window)
	windowChSize = make(chan *Window)
	windowTitle = make(chan *Window)

	events = make(chan interface{}, 32)

	ch := make(chan struct{}, 1)
	wde.BackendRun = func() {
		go sdlWindowLoop()
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

	Id int

	title string

	opdone chan struct{}

	width, height int
	keychords map[string]bool
}

type point image.Point

func NewWindow(width, height int)  (wde.Window, error) {
	fmt.Printf("new window, width %d height %d\n", width, height)
	w := new(Window)
	w.width = width
	w.height = height

	w.buffer = NewSdlBuffer(width, height)

	w.opdone = make(chan struct{})
	newWindow<-w
	<-w.opdone
	return w, nil
}

///////////////////
//Interface methods
///////////////////

func (w *Window) SetTitle(title string) {
	if w.closed {
		return
	}
	w.title = title
	windowTitle <- w
	<-w.opdone
}

func (w *Window) SetSize(width, height int) {
	if w.closed {
		return
	}
	w.width = width
	w.height = height
	windowChSize <- w
	<-w.opdone
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
	return events
}

func (w *Window) Close() error {
	w.w.Destroy()
	w.closed = true
	return nil
}

func (w *Window) FlushImage(r ...image.Rectangle) {
	if w.closed {
		return
	}
	windowFlush <- w
	<-w.opdone
}

func (w *Window) Screen() wde.Image {
	return w.buffer
}

func (w *Window) Show() {
	windowShow <- w
	<-w.opdone
}

///////////////////////
//Non interface methods
///////////////////////

//Main thread for sdl calls to be made in
func sdlWindowLoop() {
	runtime.LockOSThread()
	for {
		select {
		case w := <-newWindow:
			w.Id = len(windowList)
			windowList = append(windowList, w)
			w.setupWindow()
			w.opdone<-struct{}{}
		case w := <-windowFlush:
			for x := 0; x < w.width; x++ {
				for y := 0; y < w.height; y++ {
					r,g,b,a := w.buffer.At(x,y).RGBA()
					w.r.SetDrawColor(uint8(r),uint8(g),uint8(b),uint8(a))
					w.r.DrawPoint(x,y)
				}
			}
			w.r.Present()
			w.buffer.Clear()
			w.opdone<-struct{}{}
		case w := <-windowShow:
			w.w.Show()
			w.opdone<-struct{}{}
		case w := <-windowChSize:
			if !w.lock {
				w.w.SetSize(w.width, w.height)
			}
			w.opdone<-struct{}{}
		case w := <-windowTitle:
			w.w.SetTitle(w.title)
			w.opdone <- struct{}{}
		default:
			for collectEvents() {}
			time.Sleep(time.Millisecond * 10);
		}
	}
}

func collectEvents() bool {
	e := sdl.PollEvent()
	if e == nil {
		return false
	}
	//Event translation
	switch e := e.(type) {
	case *sdl.KeyDownEvent:
		rev := new(wde.KeyDownEvent)
		rev.Key = ConvertKeyCode(e.Keysym.Scancode)
		keychords[rev.Key] = true
		events <- rev
		chord := new(wde.KeyTypedEvent)
		chord.Chord = wde.ConstructChord(keychords)
		events <- chord
		return true
	case *sdl.KeyUpEvent:
		rev := new(wde.KeyUpEvent)
		rev.Key = ConvertKeyCode(e.Keysym.Scancode)
		keychords[rev.Key] = false
		events <- rev
		chord := new(wde.KeyTypedEvent)
		chord.Chord = wde.ConstructChord(keychords)
		events <- chord
		return true
	case *sdl.MouseButtonEvent:
		fmt.Println("Mouse button event...")
		rev := new(wde.MouseButtonEvent)
		rev.Which = wde.Button(1 << e.Button)
		log.Printf("Button: %d\n",e.Button)
		rev.Where = image.Pt(int(e.X),int(e.Y))
		events <- rev
		return true
	case *sdl.MouseMotionEvent:

		return true
	case *sdl.MouseWheelEvent:
		return true
	case *sdl.QuitEvent:
		events <- new(wde.CloseEvent)
		return true
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
			events <- new(wde.MouseEnteredEvent)
			log.Println("Mouse enter...")
		case sdl.WINDOWEVENT_LEAVE:
			events <- new(wde.MouseExitedEvent)
			log.Println("Mouse leave...")
		case sdl.WINDOWEVENT_RESIZED:
			log.Printf("resize to: %d %d\n", e.Data1, e.Data2)
			rev := new(wde.ResizeEvent)
			rev.Height = int(e.Data1)
			rev.Width = int(e.Data2)
			events <- rev
		case sdl.WINDOWEVENT_CLOSE:
			log.Println("Close the window please.")
			events <- &wde.CloseEvent{}
		case sdl.WINDOWEVENT_FOCUS_GAINED:
			log.Println("Focus gained, woot!")
		case sdl.WINDOWEVENT_FOCUS_LOST:
			log.Println("Focus lost, must have ADHD.")
		case sdl.WINDOWEVENT_MOVED:
			log.Printf("please move window to %d %d.\n", e.Data1, e.Data2)
		default:
			log.Printf("UNRECOGNIZED WINDOW EVENT: %d\n", e.Event)
		}
		return true
	}
	return false
}

func (w *Window) setupWindow() error {
	window := sdl.CreateWindow("", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, w.width, w.height, sdl.WINDOW_SHOWN)
	if window == nil {
		return sdl.GetError()
	}

	renderer := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if renderer == nil {
		return sdl.GetError()
	}

	w.w = window
	w.r = renderer
	return nil
}

/*
func (w *Window) manageThread(width, height int, ready chan error) {
	runtime.LockOSThread()

	window := sdl.CreateWindow("", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, w.width, w.height, sdl.WINDOW_SHOWN)
	if window == nil {
		fmt.Fprintf(os.Stderr, "Failed to create window: %s\n", sdl.GetError());
		os.Exit(1);
	}

	renderer := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if renderer == nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %s\n", sdl.GetError());
		os.Exit(2);
	}

	w.w = window
	w.r = renderer

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
*/

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
