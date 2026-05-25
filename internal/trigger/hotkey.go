package trigger

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procRegisterHotKey    = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32.NewProc("UnregisterHotKey")
	procCreateWindowExW   = user32.NewProc("CreateWindowExW")
	procDefWindowProcW    = user32.NewProc("DefWindowProcW")
	procGetMessageW       = user32.NewProc("GetMessageW")
	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procPostQuitMessage   = user32.NewProc("PostQuitMessage")
	procGetModuleHandleW  = kernel32.NewProc("GetModuleHandleW")
)

const (
	WM_HOTKEY = 0x0312
	MOD_ALT       = 0x0001
	MOD_CONTROL   = 0x0002
	MOD_SHIFT     = 0x0004
	MOD_WIN       = 0x0008
	MOD_NOREPEAT  = 0x4000
)

type HotkeyHandler func()

type HotkeyManager struct {
	mu       sync.Mutex
	handlers map[int]HotkeyHandler
	nextID   int
	hwnd     uintptr
	done     chan struct{}
}

func NewHotkeyManager() (*HotkeyManager, error) {
	runtime.LockOSThread()

	hinstance, _, _ := procGetModuleHandleW.Call(0)

	hwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("EvidenceGuardian_Hotkey"))),
		0, 0, 0, 0, 0, 0,
		0, 0, hinstance, 0,
	)
	if hwnd == 0 {
		return nil, syscall.GetLastError()
	}

	return &HotkeyManager{
		handlers: make(map[int]HotkeyHandler),
		hwnd:     hwnd,
		done:     make(chan struct{}),
	}, nil
}

func (m *HotkeyManager) Register(mods, vk int, handler HotkeyHandler) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++

	ret, _, _ := procRegisterHotKey.Call(m.hwnd, uintptr(id), uintptr(mods), uintptr(vk))
	if ret == 0 {
		return 0, syscall.GetLastError()
	}

	m.handlers[id] = handler
	return id, nil
}

func (m *HotkeyManager) Unregister(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	procUnregisterHotKey.Call(m.hwnd, uintptr(id))
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
		procUnregisterHotKey.Call(m.hwnd, uintptr(id))
	}
	m.handlers = nil
	m.mu.Unlock()
}
