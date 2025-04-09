package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// 默认TOTP参数
	DefaultDigits    = 6
	DefaultPeriod    = 30
	DefaultAlgorithm = "SHA1"
	DefaultIssuer    = "Mist"
)

var (
	// ErrInvalidOTP 表示OTP无效
	ErrInvalidOTP = errors.New("提供的OTP代码无效")
	// ErrInvalidSecret 表示密钥无效
	ErrInvalidSecret = errors.New("提供的密钥格式无效")
	// ErrInvalidInput 表示输入参数无效
	ErrInvalidInput = errors.New("提供的输入参数无效")
)

// TOTPConfig TOTP配置结构体
type TOTPConfig struct {
	// Digits 代码位数，通常为6
	Digits int
	// Period 刷新周期，通常为30秒
	Period int
	// Algorithm 使用的哈希算法，通常为SHA1
	Algorithm string
	// Issuer 发行者名称，通常为应用名
	Issuer string
	// SecretSize 密钥大小（字节）
	SecretSize int
}

// DefaultTOTPConfig 返回默认TOTP配置
func DefaultTOTPConfig() TOTPConfig {
	return TOTPConfig{
		Digits:     DefaultDigits,
		Period:     DefaultPeriod,
		Algorithm:  DefaultAlgorithm,
		Issuer:     DefaultIssuer,
		SecretSize: 20, // 160位
	}
}

// GenerateSecret 生成随机TOTP密钥
func GenerateSecret(size int) (string, error) {
	if size <= 0 {
		size = 20 // 默认160位
	}

	// 生成随机字节
	secret := make([]byte, size)
	_, err := rand.Read(secret)
	if err != nil {
		return "", err
	}

	// 使用Base32编码（TOTP标准）
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateTOTPCode 基于密钥和当前时间生成TOTP代码
func GenerateTOTPCode(secret string, config TOTPConfig) (string, error) {
	// 解码密钥
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", ErrInvalidSecret
	}

	// 获取当前时间步
	timeStep := uint64(time.Now().Unix() / int64(config.Period))

	// 生成TOTP代码
	return generateTOTP(secretBytes, timeStep, config)
}

// ValidateTOTPCode 验证TOTP代码
// 允许1个时间周期的误差（前后30秒）
func ValidateTOTPCode(secret, code string, config TOTPConfig) bool {
	// 解码密钥
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	// 获取当前时间步
	timeStep := uint64(time.Now().Unix() / int64(config.Period))

	// 检查当前、上一个和下一个时间步的代码
	for _, step := range []uint64{timeStep - 1, timeStep, timeStep + 1} {
		validCode, err := generateTOTP(secretBytes, step, config)
		if err != nil {
			continue
		}

		if validCode == code {
			return true
		}
	}

	return false
}

// GenerateProvisioningURI 生成TOTP配置URI
// 用于生成二维码，让用户扫码添加到验证器应用（如Google Authenticator）
func GenerateProvisioningURI(secret, accountName string, config TOTPConfig) string {
	params := url.Values{}
	params.Add("secret", secret)
	params.Add("algorithm", config.Algorithm)
	params.Add("digits", fmt.Sprintf("%d", config.Digits))
	params.Add("period", fmt.Sprintf("%d", config.Period))

	if config.Issuer != "" {
		params.Add("issuer", config.Issuer)
		accountName = fmt.Sprintf("%s:%s", config.Issuer, accountName)
	}

	return fmt.Sprintf("otpauth://totp/%s?%s", url.QueryEscape(accountName), params.Encode())
}

// 内部函数，生成TOTP代码
func generateTOTP(secret []byte, timeStep uint64, config TOTPConfig) (string, error) {
	// 将时间步转换为字节数组
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, timeStep)

	// 计算HMAC-SHA1
	hash := hmac.New(sha1.New, secret)
	_, err := hash.Write(msg)
	if err != nil {
		return "", err
	}

	// 获取哈希结果
	h := hash.Sum(nil)

	// 动态截断
	offset := h[len(h)-1] & 0xf
	binary := (uint32(h[offset]&0x7f)<<24 |
		uint32(h[offset+1])<<16 |
		uint32(h[offset+2])<<8 |
		uint32(h[offset+3])) % uint32(pow10(config.Digits))

	// 格式化结果为固定长度字符串
	format := fmt.Sprintf("%%0%dd", config.Digits)
	return fmt.Sprintf(format, binary), nil
}

// 计算10的n次方
func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}

// TOTP 是一个简化使用的TOTP结构体
type TOTP struct {
	Secret string
	Config TOTPConfig
}

// NewTOTP 创建一个新的TOTP实例
func NewTOTP(config ...TOTPConfig) (*TOTP, error) {
	cfg := DefaultTOTPConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	secret, err := GenerateSecret(cfg.SecretSize)
	if err != nil {
		return nil, err
	}

	return &TOTP{
		Secret: secret,
		Config: cfg,
	}, nil
}

// NewTOTPWithSecret 使用已有密钥创建TOTP实例
func NewTOTPWithSecret(secret string, config ...TOTPConfig) *TOTP {
	cfg := DefaultTOTPConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &TOTP{
		Secret: secret,
		Config: cfg,
	}
}

// Generate 生成当前TOTP代码
func (t *TOTP) Generate() (string, error) {
	return GenerateTOTPCode(t.Secret, t.Config)
}

// Validate 验证TOTP代码
func (t *TOTP) Validate(code string) bool {
	return ValidateTOTPCode(t.Secret, code, t.Config)
}

// ProvisioningURI 生成配置URI
func (t *TOTP) ProvisioningURI(accountName string) string {
	return GenerateProvisioningURI(t.Secret, accountName, t.Config)
}
