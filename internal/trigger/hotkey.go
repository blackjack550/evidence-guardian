package trigger

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
)

const (
	WM_HOTKEY    = 0x0312
	MOD_ALT      = 0x0001
	MOD_CONTROL  = 0x0002
	MOD_SHIFT    = 0x0004
	MOD_WIN      = 0x0008
	MOD_NOREPEAT = 0x4000
)

type HotkeyHandler func()

type HotkeyManager struct {
	mu       sync.Mutex
	handlers map[int]HotkeyHandler
	nextID   int
	done     chan struct{}
}

func NewHotkeyManager() (*HotkeyManager, error) {
	runtime.LockOSThread()
	// RegisterHotKey with HWND=0 posts WM_HOTKEY to this thread's message queue
	return &HotkeyManager{
		handlers: make(map[int]HotkeyHandler),
		done:     make(chan struct{}),
	}, nil
}

func (m *HotkeyManager) Register(mods, vk int, handler HotkeyHandler) (int, error) {
	id := m.nextID
	m.nextID++

	// HWND=0 means messages go to this thread's queue
	ret, _, _ := procRegisterHotKey.Call(0, uintptr(id), uintptr(mods), uintptr(vk))
	if ret == 0 {
		return 0, syscall.GetLastError()
	}

	m.mu.Lock()
	m.handlers[id] = handler
	m.mu.Unlock()
	return id, nil
}

func (m *HotkeyManager) Unregister(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	procUnregisterHotKey.Call(0, uintptr(id))
	delete(m.handlers, id)
}

func (m *HotkeyManager) Start() {
	var msg struct {
		hwnd    uintptr
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      struct{ x, y int32 }
	}

	for {
		ret, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&msg)), 0, 0, 0,
		)
		if ret == 0 {
			close(m.done)
			return
		}

		if msg.message == WM_HOTKEY {
			id := int(msg.wParam)
			m.mu.Lock()
			handler, ok := m.handlers[id]
			m.mu.Unlock()
			if ok && handler != nil {
				handler()
			}
		}
	}
}

func (m *HotkeyManager) Stop() {
	procPostQuitMessage.Call(0)
	<-m.done

	m.mu.Lock()
	for id := range m.handlers {
		procUnregisterHotKey.Call(0, uintptr(id))
	}
	m.handlers = nil
	m.mu.Unlock()
}
