//go:build !linux
// +build !linux

package mist

import (
	"io"
	"net"
	"os"
)

// sendFileImpl 是非Linux平台上的实现（回退到标准IO）
func sendFileImpl(filefd int, conn *net.TCPConn, size int64) error {
	// 非Linux平台使用标准IO复制
	_, err := io.Copy(conn, os.NewFile(uintptr(filefd), ""))
	return err
}
