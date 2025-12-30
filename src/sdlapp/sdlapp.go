package sdlapp

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

func CreateWindow(props ...func(props sdl.PropertiesID) error) (*sdl.Window, error) {
	router.mu.Lock()
	defer router.mu.Unlock()

	window, err := sdl.CreateWindow(props...)
	if err != nil {
		return nil, err
	}

	if router.primaryWindow == 0 {
		router.primaryWindow = window.ID()
		close(router.inited)
	}
	router.windows[window.ID()] = make(chan sdl.Event)

	return window, nil
}

func Event(w *sdl.Window) sdl.Event {
	router.mu.Lock()
	ch := router.windows[w.ID()]
	router.mu.Unlock()
	if ch == nil {
		panic("guh")
	}
	return <-ch
}

func Main() error {
	<-router.inited

eventLoop:
	for {
		e, err := sdl.WaitEvent()
		if err != nil {
			return err
		}

		windowID := router.primaryWindow
		switch e := e.(type) {
		case *sdl.QuitEvent:
			break eventLoop
		case *sdl.WindowPixelSizeChangedEvent:
			windowID = e.WindowID
		case *sdl.KeyDownEvent:
			windowID = e.WindowID
		case *sdl.KeyUpEvent:
			windowID = e.WindowID
		case *sdl.MouseMotionEvent:
			windowID = e.WindowID
		case *sdl.MouseButtonDownEvent:
			windowID = e.WindowID
		case *sdl.MouseButtonUpEvent:
			windowID = e.WindowID
		}

		router.mu.Lock()
		ch := router.windows[windowID]
		router.mu.Unlock()
		if ch == nil {
			panic("guh")
		}
		ch <- e
	}

	return nil
}
