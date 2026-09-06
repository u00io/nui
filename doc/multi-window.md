# Multiple windows and threading rules

## Running several windows

```go
win1 := nui.CreateWindow("Window 1", 100, 100, 500, 300, true, false)
win2 := nui.CreateWindow("Window 2", 650, 100, 500, 300, true, false)

nui.Run(win1, win2) // blocks until ALL windows are closed
```

`nui.Run` starts one goroutine per window (`go win.Exec()`) and waits for all to finish. Equivalent manual form:

```go
go win1.Exec()
go win2.Exec()
// ... wait yourself, e.g. sync.WaitGroup
```

Opening a window later (e.g. from a button click) works the same way:

```go
win.OnKeyDown(func(k nuikey.Key, m nuikey.KeyModifiers) bool {
	if k == nuikey.KeyN {
		other := nui.CreateWindow("New window", 0, 0, 300, 150, true, false)
		other.OnPaint(...)
		go other.Exec()
	}
	return true
})
```

## Modal dialogs

```go
dlg.ShowModal(parentWin)
```

Internally runs `go dlg.Exec()`. Non-blocking: `parentWin`'s own event loop, timers and repaint keep running. The OS window manager blocks mouse/keyboard input to `parentWin` while `dlg` is open (GNOME/KDE; not enforced by all window managers, e.g. openbox).

## Threading rules (must follow)

1. **One goroutine per window.** Never call `Exec()`/`EventLoop()`/`Show()` for the same window twice or from two goroutines at once.
2. **Call a window's own methods/setters only from that window's own callbacks** (its `On*` handlers), or before its `Exec()`/goroutine has started. Do not call `win2.SetTitle(...)`, `win2.Resize(...)`, etc. from inside `win1`'s callback.
3. **Exception: `Close()`.** `win.Close()` is safe to call from any goroutine, including another window's callback (e.g. a parent closing a child dialog). It only requests the close; the actual teardown always runs on the window's own event-loop goroutine.
4. **Don't block inside callbacks** (`OnPaint`, `OnKeyDown`, `OnTimer`, ...). A blocked callback freezes that window's repaint/timer/input until it returns. Long work belongs on its own goroutine; hand results back via a channel and call `win.Update()`.
5. **Shared state read/written from multiple windows' callbacks needs your own synchronization** (mutex/channel) — the library does not add any for you.

## Why this works safely on Linux (X11)

- Each window opens its own X11 `Display` connection; `XInitThreads()` is called once at startup, so Xlib supports being driven from independent goroutines/threads.
- Each window has its own paint buffer — no shared global framebuffer between windows.
- `Close()` defers the actual `XDestroyWindow`/`XCloseDisplay` to the window's own event-loop goroutine, so a cross-window `Close()` call can never race with that window's own event pump.
