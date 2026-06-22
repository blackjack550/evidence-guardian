package capture

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procEnumWindows        = user32.NewProc("EnumWindows")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procGetClassNameW      = user32.NewProc("GetClassNameW")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW    = kernel32.NewProc("Process32FirstW")
	procProcess32NextW     = kernel32.NewProc("Process32NextW")
)

type WindowInfo struct {
	HWND        uintptr
	Title       string
	ClassName   string
	ProcessID   uint32
	ProcessName string
	Rect        RECT
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func EnumWindows() ([]WindowInfo, error) {
	// Build process name map ONCE to avoid per-window snapshot overhead
	procNames := buildProcessMap()

	var windows []WindowInfo

	callback := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}

		var titleBuf [512]uint16
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&titleBuf[0])), 512)
		title := syscall.UTF16ToString(titleBuf[:])

		var classBuf [256]uint16
		procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&classBuf[0])), 256)
		className := syscall.UTF16ToString(classBuf[:])

		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

		var rect RECT
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

		windows = append(windows, WindowInfo{
			HWND:        hwnd,
			Title:       title,
			ClassName:   className,
			ProcessID:   pid,
			ProcessName: procNames[pid],
			Rect:        rect,
		})
		return 1
	})

	procEnumWindows.Call(callback, 0)
	return windows, nil
}

func buildProcessMap() map[uint32]string {
	m := make(map[uint32]string)
	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(0x00000002, 0)
	if snapshot == 0 || snapshot == uintptr(^uint32(0)) {
		return m
	}
	defer syscall.CloseHandle(syscall.Handle(snapshot))

	var entry _PROCESSENTRY32W
	entry.dwSize = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for ret != 0 {
		pid := entry.th32ProcessID
		name := syscall.UTF16ToString(entry.szExeFile[:])
		m[pid] = name
		ret, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	return m
}

func FindTargetWindows(targets []AppTarget) ([]WindowInfo, error) {
	all, err := EnumWindows()
	if err != nil {
		return nil, err
	}

	targetMap := make(map[string]AppTarget)
	for _, t := range targets {
		targetMap[t.WindowClass] = t
	}

	var result []WindowInfo
	for _, w := range all {
		if t, ok := targetMap[w.ClassName]; ok && t.Enabled && w.Title != "" {
			result = append(result, w)
		}
	}
	return result, nil
}

type _PROCESSENTRY32W struct {
	dwSize              uint32
	cntUsage            uint32
	th32ProcessID       uint32
	th32DefaultHeapID   uintptr
	th32ModuleID        uint32
	cntThreads          uint32
	th32ParentProcessID uint32
	pcPriClassBase      int32
	dwFlags             uint32
	szExeFile           [260]uint16
}

func (w WindowInfo) String() string {
	return fmt.Sprintf("[%s] %s (PID:%d) %dx%d",
		w.ClassName, w.Title, w.ProcessID,
		w.Rect.Right-w.Rect.Left, w.Rect.Bottom-w.Rect.Top)
}

type AppTarget struct {
	Name        string
	Process     string
	WindowClass string
	Enabled     bool
}
