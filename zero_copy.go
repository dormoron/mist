package mist

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ZeroCopyResponse 提供高效的零拷贝响应机制
// 这个结构体设计用于大文件传输场景，可以避免不必要的内存拷贝，
// 直接将文件内容从磁盘传输到网络连接
type ZeroCopyResponse struct {
	// 内部状态
	writer     http.ResponseWriter
	statusCode int
	size       int64
	written    bool

	// 是否支持零拷贝传输
	zeroCopySupported bool
}

// NewZeroCopyResponse 创建一个新的零拷贝响应对象
func NewZeroCopyResponse(w http.ResponseWriter) *ZeroCopyResponse {
	return &ZeroCopyResponse{
		writer:            w,
		statusCode:        http.StatusOK,
		zeroCopySupported: checkZeroCopySupport(w),
	}
}

// checkZeroCopySupport 检查底层连接是否支持零拷贝传输
func checkZeroCopySupport(w http.ResponseWriter) bool {
	// 尝试获取底层TCP连接
	if hijacker, ok := w.(http.Hijacker); ok {
		conn, _, err := hijacker.Hijack()
		if err == nil {
			// 恢复连接
			defer conn.Close()

			// 检查是否可以转换为*net.TCPConn
			if _, ok := conn.(*net.TCPConn); ok {
				// 底层是TCP连接，可能支持零拷贝
				return true
			}
		}
	}
	return false
}

// Header 返回响应头
func (z *ZeroCopyResponse) Header() http.Header {
	return z.writer.Header()
}

// WriteHeader 写入状态码
func (z *ZeroCopyResponse) WriteHeader(statusCode int) {
	if !z.written {
		z.statusCode = statusCode
		z.writer.WriteHeader(statusCode)
		z.written = true
	}
}

// Write 实现标准的Write接口
func (z *ZeroCopyResponse) Write(data []byte) (int, error) {
	if !z.written {
		z.WriteHeader(z.statusCode)
	}
	n, err := z.writer.Write(data)
	z.size += int64(n)
	return n, err
}

// ServeFile 使用零拷贝技术提供文件服务
// 如果底层连接支持sendfile系统调用，将使用零拷贝方式
// 否则回退到标准http.ServeContent
func (z *ZeroCopyResponse) ServeFile(filePath string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	// 设置Content-Type和Content-Length头
	if z.Header().Get("Content-Type") == "" {
		z.Header().Set("Content-Type", detectContentType(filePath))
	}
	z.Header().Set("Content-Length", itoa(fileInfo.Size()))

	// 处理范围请求
	var rangeReq *rangeRequest
	if reqRange := z.Header().Get("Range"); reqRange != "" {
		rangeReq = parseRange(reqRange, fileInfo.Size())
		if rangeReq != nil {
			z.statusCode = http.StatusPartialContent
			z.Header().Set("Content-Range", rangeReq.contentRange)
		}
	}

	// 如果支持零拷贝且没有范围请求，尝试使用sendfile
	if z.zeroCopySupported && rangeReq == nil {
		return z.sendFileZeroCopy(file, fileInfo.Size())
	}

	// 否则使用标准方式
	z.WriteHeader(z.statusCode)

	// 处理范围请求
	if rangeReq != nil {
		if _, err := file.Seek(rangeReq.start, io.SeekStart); err != nil {
			return err
		}
		if _, err := io.CopyN(z.writer, file, rangeReq.length); err != nil {
			return err
		}
		return nil
	}

	// 完整文件传输
	_, err = io.Copy(z.writer, file)
	return err
}

// sendFileZeroCopy 使用sendfile系统调用实现零拷贝
func (z *ZeroCopyResponse) sendFileZeroCopy(file *os.File, size int64) error {
	// 写入响应头
	z.WriteHeader(z.statusCode)

	// 获取底层连接
	if hijacker, ok := z.writer.(http.Hijacker); ok {
		conn, _, err := hijacker.Hijack()
		if err != nil {
			// 回退到普通传输
			_, err = io.Copy(z.writer, file)
			return err
		}
		defer conn.Close()

		// 获取文件和连接的文件描述符
		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			// 回退到普通传输
			_, err = io.Copy(conn, file)
			return err
		}

		// 使用sendfile系统调用
		return sendFileFd(file, tcpConn, size)
	}

	// 不支持Hijack，回退到普通传输
	_, err := io.Copy(z.writer, file)
	return err
}

// sendFileFd 使用sendfile系统调用
func sendFileFd(file *os.File, conn *net.TCPConn, size int64) error {
	// 调用平台特定实现
	return sendFileImpl(int(file.Fd()), conn, size)
}

// 范围请求结构
type rangeRequest struct {
	start        int64
	length       int64
	contentRange string
}

// parseRange 解析Range请求头
func parseRange(rangeHeader string, fileSize int64) *rangeRequest {
	// 简化实现，仅处理单一范围
	const prefix = "bytes="
	if !strings.HasPrefix(rangeHeader, prefix) {
		return nil
	}

	rangeStr := strings.TrimPrefix(rangeHeader, prefix)
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return nil
	}

	var start, end int64
	var err error

	// 解析起始位置
	if parts[0] == "" {
		// -N 表示最后N个字节
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil
		}
		start = fileSize - end
		if start < 0 {
			start = 0
		}
		end = fileSize - 1
	} else {
		// M-N 表示从M字节到N字节
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil
		}

		if parts[1] == "" {
			// M- 表示从M字节到文件结尾
			end = fileSize - 1
		} else {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil
			}
		}
	}

	// 验证范围是否有效
	if start >= fileSize || end >= fileSize || start > end {
		return nil
	}

	length := end - start + 1
	contentRange := fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize)

	return &rangeRequest{
		start:        start,
		length:       length,
		contentRange: contentRange,
	}
}

// detectContentType 根据文件扩展名检测Content-Type
func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	// 常见MIME类型映射
	mimeTypes := map[string]string{
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".mp3":  "audio/mpeg",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".webp": "image/webp",
		".zip":  "application/zip",
		".ico":  "image/x-icon",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}

	// 默认二进制流
	return "application/octet-stream"
}

// itoa 是一个用于数字到字符串转换的帮助函数
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
