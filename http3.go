package mist

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// HTTP3Server 提供HTTP/3服务器功能
type HTTP3Server struct {
	httpServer   *http.Server
	quicServer   *http3.Server
	quicListener *quic.EarlyListener
	log          Logger
	config       *HTTP3Config
}

// HTTP3Config 定义HTTP/3服务器的配置选项
type HTTP3Config struct {
	// QUIC配置
	MaxIdleTimeout        time.Duration
	MaxIncomingStreams    int64
	MaxIncomingUniStreams int64

	// TLS配置
	TLSConfig *tls.Config

	// 连接配置
	EnableDatagrams      bool
	HandshakeIdleTimeout time.Duration

	// Alt-Svc配置
	AltSvcHeader       string
	EnableAltSvcHeader bool
}

// DefaultHTTP3Config 返回默认的HTTP/3配置
func DefaultHTTP3Config() *HTTP3Config {
	return &HTTP3Config{
		MaxIdleTimeout:        30 * time.Second,
		MaxIncomingStreams:    100,
		MaxIncomingUniStreams: 100,
		EnableDatagrams:       false,
		HandshakeIdleTimeout:  10 * time.Second,
		EnableAltSvcHeader:    true,
		AltSvcHeader:          `h3=":443"; ma=2592000`,
	}
}

// NewHTTP3Server 创建新的HTTP/3服务器
func NewHTTP3Server(httpServer *http.Server, config *HTTP3Config, logger Logger) *HTTP3Server {
	if config == nil {
		config = DefaultHTTP3Config()
	}

	if logger == nil {
		// 使用默认日志记录器
		logger = GetDefaultLogger()
	}

	return &HTTP3Server{
		httpServer: httpServer,
		log:        logger,
		config:     config,
	}
}

// ListenAndServe 在指定地址启动HTTP/3服务器
func (s *HTTP3Server) ListenAndServe(addr string) error {
	s.log.Info("HTTP/3服务器准备启动于: %s", addr)

	// 必须有TLS配置
	if s.httpServer.TLSConfig == nil {
		return fmt.Errorf("HTTP/3需要TLS配置")
	}

	// 确保TLS配置支持QUIC
	s.httpServer.TLSConfig.NextProtos = append(s.httpServer.TLSConfig.NextProtos, "h3")

	// 创建QUIC配置
	quicConfig := &quic.Config{
		MaxIdleTimeout:        s.config.MaxIdleTimeout,
		MaxIncomingStreams:    s.config.MaxIncomingStreams,
		MaxIncomingUniStreams: s.config.MaxIncomingUniStreams,
		EnableDatagrams:       s.config.EnableDatagrams,
		HandshakeIdleTimeout:  s.config.HandshakeIdleTimeout,
	}

	// 创建HTTP/3服务器
	s.quicServer = &http3.Server{
		Handler:    s.httpServer.Handler,
		TLSConfig:  s.httpServer.TLSConfig,
		QuicConfig: quicConfig,
	}

	// 在响应头中添加Alt-Svc
	if s.config.EnableAltSvcHeader {
		// 使用中间件添加Alt-Svc头
		originalHandler := s.httpServer.Handler
		s.httpServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Alt-Svc", s.config.AltSvcHeader)
			originalHandler.ServeHTTP(w, r)
		})
	}

	// 创建QUIC监听器
	listener, err := quic.ListenAddrEarly(addr, s.httpServer.TLSConfig, quicConfig)
	if err != nil {
		return fmt.Errorf("启动QUIC监听器失败: %v", err)
	}

	s.quicListener = listener
	s.log.Info("HTTP/3服务器已启动于: %s", addr)

	return s.quicServer.ServeListener(listener)
}

// ListenAndServeTLS 使用TLS证书启动HTTP/3服务器
func (s *HTTP3Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	// 加载TLS证书
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("加载TLS证书失败: %v", err)
	}

	// 创建TLS配置
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	// 设置服务器TLS配置
	s.httpServer.TLSConfig = tlsConfig

	return s.ListenAndServe(addr)
}

// Shutdown 优雅关闭HTTP/3服务器
func (s *HTTP3Server) Shutdown(ctx context.Context) error {
	var err error

	// 关闭QUIC监听器
	if s.quicListener != nil {
		err = s.quicListener.Close()
		if err != nil {
			s.log.Error("关闭QUIC监听器失败: %v", err)
		}
	}

	// 关闭HTTP/3服务器
	if s.quicServer != nil {
		err = s.quicServer.CloseGracefully(0)
	}

	return err
}
