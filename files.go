package mist

import (
	"github.com/hashicorp/golang-lru"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FileUploader struct {
	// FileField 对应于文件在表单中的字段名字
	FileField string
	// FileField 对应于文件在表单中的字段名字
	DstPathFunc func(*multipart.FileHeader) string
}

// Handle 文件上传
func (f *FileUploader) Handle() HandleFunc {
	return func(ctx *Context) {
		file, fileHeader, err := ctx.Request.FormFile(f.FileField)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		defer file.Close()
		dst := f.DstPathFunc(fileHeader)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o666)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		defer dstFile.Close()
		_, err = io.CopyBuffer(dstFile, file, nil)
		if err != nil {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Upload failure" + err.Error())
			return
		}
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = []byte("Upload success")
	}
}

type FileDownloader struct {
	Dir string
}

// Handle 文件下载
func (f *FileDownloader) Handle() HandleFunc {
	return func(ctx *Context) {
		req := ctx.QueryValue("file")
		if req.err != nil {
			ctx.RespStatusCode = http.StatusBadRequest
			ctx.RespData = []byte("The destination file could not be found")
			return
		}
		req.val = filepath.Clean(req.val)
		dst := filepath.Join(f.Dir, req.val)
		dst, req.err = filepath.Abs(dst)
		if !strings.Contains(dst, f.Dir) {
			ctx.RespStatusCode = http.StatusBadRequest
			ctx.RespData = []byte("Access path error")
			return
		}
		fn := filepath.Base(dst)
		header := ctx.ResponseWriter.Header()
		header.Set("Content-Disposition", "attachment;filename="+fn)
		header.Set("Content-Description", "File Transfer")
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Transfer-Encoding", "binary")
		header.Set("Expires", "0")
		header.Set("Cache-Control", "must-revalidate")
		header.Set("Pragma", "public")
		http.ServeFile(ctx.ResponseWriter, ctx.Request, dst)
	}
}

type StaticResourceHandlerOption func(handler *StaticResourceHandler)

type StaticResourceHandler struct {
	dir               string
	cache             *lru.Cache
	extContentTypeMap map[string]string
	maxSize           int
}

func InitStaticResourceHandler(dir string, opts ...StaticResourceHandlerOption) (*StaticResourceHandler, error) {
	c, err := lru.New(1000)
	if err != nil {
		return nil, err
	}
	res := &StaticResourceHandler{
		dir:     dir,
		cache:   c,
		maxSize: 1024 * 1024,
		extContentTypeMap: map[string]string{
			"jpeg": "image/jpeg",
			"jpe":  "image/jpeg",
			"jpg":  "image/jpeg",
			"png":  "image/png",
			"pdf":  "image/pdf",
		},
	}
	for _, opt := range opts {
		opt(res)
	}
	return res, nil
}

func StaticWithMaxFileSize(maxSize int) StaticResourceHandlerOption {
	return func(handler *StaticResourceHandler) {
		handler.maxSize = maxSize
	}
}

func StaticWithCache(c *lru.Cache) StaticResourceHandlerOption {
	return func(handler *StaticResourceHandler) {
		handler.cache = c
	}
}

func StaticWithExtension(extMap map[string]string) StaticResourceHandlerOption {
	return func(handler *StaticResourceHandler) {
		for extMap, contentType := range extMap {
			handler.extContentTypeMap[extMap] = contentType
		}
	}
}

func (s *StaticResourceHandler) Handle(ctx *Context) {
	file := ctx.PathValue("file")
	if file.err != nil {
		ctx.RespStatusCode = http.StatusBadRequest
		ctx.RespData = []byte("Request path error")
		return
	}
	dst := filepath.Join(s.dir, file.val)
	ext := filepath.Ext(dst)[1:]
	header := ctx.ResponseWriter.Header()
	if data, ok := s.cache.Get(file.val); ok {
		header.Set("Content-Type", s.extContentTypeMap[ext])
		header.Set("Content-Length", strconv.Itoa(len(data.([]byte))))
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = data.([]byte)
		return
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		ctx.RespStatusCode = http.StatusInternalServerError
		ctx.RespData = []byte("Server error")
		return
	}

	if len(data) <= s.maxSize {
		s.cache.Add(file.val, data)
	}
	header.Set("Content-Type", s.extContentTypeMap[ext])
	header.Set("Content-Length", strconv.Itoa(len(data)))
	ctx.RespStatusCode = http.StatusOK
	ctx.RespData = data
}
