//go:build windows

package fswatch

import (
	"context"
	"sync"
	"syscall"
)

// Watch blocks in Windows directory notifications until a change or cancellation.
// Each watched directory owns one handle; notify may run on a worker goroutine.
func Watch(ctx context.Context, dirs []string, notify func(), report func(error)) error {
	var handles []syscall.Handle
	for _, dir := range dirs {
		path, err := syscall.UTF16PtrFromString(dir)
		if err != nil {
			for _, opened := range handles {
				_ = syscall.CloseHandle(opened)
			}
			return err
		}
		handle, err := syscall.CreateFile(path, syscall.FILE_LIST_DIRECTORY,
			syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
			nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
		if err != nil {
			for _, opened := range handles {
				_ = syscall.CloseHandle(opened)
			}
			return err
		}
		handles = append(handles, handle)
	}
	var once sync.Once
	closeHandles := func() {
		once.Do(func() {
			for _, handle := range handles {
				_ = syscall.CancelIoEx(handle, nil)
				_ = syscall.CloseHandle(handle)
			}
		})
	}
	go func() { <-ctx.Done(); closeHandles() }()
	for _, handle := range handles {
		go func(handle syscall.Handle) {
			var buffer [64 << 10]byte
			for ctx.Err() == nil {
				var count uint32
				err := syscall.ReadDirectoryChanges(handle, &buffer[0], uint32(len(buffer)), true,
					syscall.FILE_NOTIFY_CHANGE_FILE_NAME|syscall.FILE_NOTIFY_CHANGE_DIR_NAME|
						syscall.FILE_NOTIFY_CHANGE_LAST_WRITE|syscall.FILE_NOTIFY_CHANGE_SIZE,
					&count, nil, 0)
				if err != nil {
					if ctx.Err() == nil {
						report(err)
					}
					return
				}
				if ctx.Err() == nil {
					notify()
				}
			}
		}(handle)
	}
	return nil
}
