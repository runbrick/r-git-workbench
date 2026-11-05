//go:build linux

package fswatch

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const watchMask = syscall.IN_CREATE | syscall.IN_MOVED_TO | syscall.IN_DELETE |
	syscall.IN_MOVED_FROM | syscall.IN_MODIFY | syscall.IN_CLOSE_WRITE |
	syscall.IN_ATTRIB | syscall.IN_DELETE_SELF | syscall.IN_MOVE_SELF

// Watch registers all existing directories and adds new directories as they
// arrive. Epoll waits on inotify and a cancellation pipe, without idle polling.
func Watch(ctx context.Context, dirs []string, notify func(), report func(error)) error {
	fd, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		return err
	}
	watchPaths := make(map[int]string)
	seen := make(map[string]bool)
	addTree := func(root string) error {
		return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() || seen[path] {
				return nil
			}
			wd, err := syscall.InotifyAddWatch(fd, path, watchMask|syscall.IN_ONLYDIR|syscall.IN_DONT_FOLLOW)
			if err != nil {
				return err
			}
			seen[path], watchPaths[wd] = true, path
			return nil
		})
	}
	for _, dir := range dirs {
		if err := addTree(dir); err != nil {
			_ = syscall.Close(fd)
			return err
		}
	}
	wakeRead, wakeWrite, err := os.Pipe()
	if err != nil {
		_ = syscall.Close(fd)
		return err
	}
	epoll, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		_ = wakeRead.Close()
		_ = wakeWrite.Close()
		_ = syscall.Close(fd)
		return err
	}
	for _, descriptor := range []int{fd, int(wakeRead.Fd())} {
		if err := syscall.EpollCtl(epoll, syscall.EPOLL_CTL_ADD, descriptor, &syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(descriptor)}); err != nil {
			_ = syscall.Close(epoll)
			_ = wakeRead.Close()
			_ = wakeWrite.Close()
			_ = syscall.Close(fd)
			return err
		}
	}
	go func() { <-ctx.Done(); _ = wakeWrite.Close() }()
	go func() {
		defer syscall.Close(fd)
		defer syscall.Close(epoll)
		defer wakeRead.Close()
		var events [2]syscall.EpollEvent
		var buffer [64 << 10]byte
		for ctx.Err() == nil {
			count, err := syscall.EpollWait(epoll, events[:], -1)
			if err == syscall.EINTR {
				continue
			}
			if err != nil {
				if ctx.Err() == nil {
					report(fmt.Errorf("Linux directory watcher: %w", err))
				}
				return
			}
			for _, event := range events[:count] {
				if int(event.Fd) != fd {
					return
				}
				for {
					n, err := syscall.Read(fd, buffer[:])
					if errors.Is(err, syscall.EAGAIN) {
						break
					}
					if err != nil || n == 0 {
						if ctx.Err() == nil {
							report(fmt.Errorf("Linux directory watcher: %v", err))
						}
						return
					}
					for offset := 0; offset+syscall.SizeofInotifyEvent <= n; {
						wd := int(int32(binary.NativeEndian.Uint32(buffer[offset:])))
						mask := binary.NativeEndian.Uint32(buffer[offset+4:])
						nameLen := int(binary.NativeEndian.Uint32(buffer[offset+12:]))
						end := offset + syscall.SizeofInotifyEvent + nameLen
						if end > n {
							return
						}
						if mask&syscall.IN_IGNORED != 0 {
							delete(seen, watchPaths[wd])
							delete(watchPaths, wd)
						}
						if mask&syscall.IN_Q_OVERFLOW != 0 {
							report(errors.New("Linux directory watcher overflowed; refresh manually"))
						}
						if mask&syscall.IN_ISDIR != 0 && mask&(syscall.IN_CREATE|syscall.IN_MOVED_TO) != 0 {
							name := buffer[offset+syscall.SizeofInotifyEvent : end]
							for i, b := range name {
								if b == 0 {
									name = name[:i]
									break
								}
							}
							if parent := watchPaths[wd]; parent != "" {
								if err := addTree(filepath.Join(parent, string(name))); err != nil {
									report(fmt.Errorf("Linux directory watcher could not register %s: %w", filepath.Join(parent, string(name)), err))
								}
							}
						}
						offset = end
					}
					if ctx.Err() == nil {
						notify()
					}
				}
			}
		}
	}()
	return nil
}
