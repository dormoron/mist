//go:build linux
// +build linux

package mist

import (
	"net"
	"syscall"
)

// sendFileImpl 是Linux平台上的sendfile系统调用实现
func sendFileImpl(filefd int, conn *net.TCPConn, size int64) error {
	// 获取连接的文件描述符
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var sendErr error

	// 执行系统调用
	rawConn.Write(func(fd uintptr) bool {
		// Linux sendfile系统调用
		n, err := syscall.Sendfile(int(fd), filefd, nil, int(size))
		if err != nil {
			sendErr = err
			return true
		}

		// 部分传输的情况
		if n < int(size) {
			size -= int64(n)
			return false // 继续传输
		}

		return true // 传输完成
	})

	return sendErr
}
