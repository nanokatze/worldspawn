package sdlrouter

import (
	"sync"

	"worldspawn/sdl"
)

// TODO: flatten into globals
type sdlrouter struct {
	inited        chan struct{}
	mu            sync.Mutex
	primaryWindow sdl.WindowID
	windows       map[sdl.WindowID]chan sdl.Event
}

var router = sdlrouter{
	inited:  make(chan struct{}),
	windows: make(map[sdl.WindowID]chan sdl.Event),
}

// TODO: wrap sdl.Window? It would let us to make Event into a method. If so,
// how we should we wrap it, by newtyping it, by creating a new struct embedding
// *sdl.Window, or by creating a new struct with private *sdl.Window member?

func CreateWindow(props ...func(props sdl.PropertiesID) error) *sdl.Window {
	router.mu.Lock()
	defer router.mu.Unlock()

	window, err := sdl.CreateWindow(props...)
	if err != nil {
		panic(err)
	}

	if router.primaryWindow == 0 {
		router.primaryWindow = window.ID()
		close(router.inited)
	}
	router.windows[window.ID()] = make(chan sdl.Event)

	return window
}

func Events(w *sdl.Window) <-chan sdl.Event {
	router.mu.Lock()
	ch := router.windows[w.ID()]
	router.mu.Unlock()
	if ch == nil {
		panic("guh")
	}
	return ch
}

func Main() {
	<-router.inited

eventLoop:
	for {
		event, err := sdl.WaitEvent()
		if err != nil {
			panic(err)
		}

		router.mu.Lock()
		windowID := router.primaryWindow
		switch event := event.(type) {
		case *sdl.QuitEvent:
			router.mu.Unlock()
			break eventLoop
		case *sdl.WindowPixelSizeChangedEvent:
			windowID = event.WindowID
		case *sdl.KeyDownEvent:
			windowID = event.WindowID
		case *sdl.KeyUpEvent:
			windowID = event.WindowID
		case *sdl.MouseMotionEvent:
			windowID = event.WindowID
		case *sdl.MouseButtonDownEvent:
			windowID = event.WindowID
		case *sdl.MouseButtonUpEvent:
			windowID = event.WindowID
		}
		ch := router.windows[windowID]
		router.mu.Unlock()
		if ch == nil {
			panic("guh")
		}
		ch <- event
	}
}
