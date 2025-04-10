package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// 默认TOTP参数
	DefaultDigits    = 6
	DefaultPeriod    = 30
	DefaultAlgorithm = "SHA1"
	DefaultIssuer    = "Mist"
	// 备份码长度
	DefaultBackupCodeLength = 8
	// 默认备份码数量
	DefaultBackupCodeCount = 10
	// 默认验证窗口大小（允许前后多少个时间单位）
	DefaultWindowSize = 1
)

// 支持的算法
const (
	AlgorithmSHA1   = "SHA1"
	AlgorithmSHA256 = "SHA256"
	AlgorithmSHA512 = "SHA512"
)

var (
	// ErrInvalidOTP 表示OTP无效
	ErrInvalidOTP = errors.New("提供的OTP代码无效")
	// ErrInvalidSecret 表示密钥无效
	ErrInvalidSecret = errors.New("提供的密钥格式无效")
	// ErrInvalidInput 表示输入参数无效
	ErrInvalidInput = errors.New("提供的输入参数无效")
	// ErrInvalidAlgorithm 表示算法无效
	ErrInvalidAlgorithm = errors.New("提供的哈希算法无效")
	// ErrAllBackupCodesUsed 表示所有备份码都已使用
	ErrAllBackupCodesUsed = errors.New("所有备份码都已使用")
	// ErrBackupCodeInvalid 表示备份码无效
	ErrBackupCodeInvalid = errors.New("提供的备份码无效")
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
	// WindowSize 验证窗口大小，即允许前后多少个时间单位
	WindowSize int
	// SkipValidUsedTokens 是否跳过验证已使用过的令牌（防止重放攻击）
	SkipValidUsedTokens bool
}

// BackupCode 备份码结构体
type BackupCode struct {
	// Code 备份码
	Code string
	// Used 是否已使用
	Used bool
}

// DefaultTOTPConfig 返回默认TOTP配置
func DefaultTOTPConfig() TOTPConfig {
	return TOTPConfig{
		Digits:              DefaultDigits,
		Period:              DefaultPeriod,
		Algorithm:           DefaultAlgorithm,
		Issuer:              DefaultIssuer,
		SecretSize:          20, // 160位
		WindowSize:          DefaultWindowSize,
		SkipValidUsedTokens: true,
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

// 获取哈希算法
func getHashFunc(algorithm string) (func() hash.Hash, error) {
	switch strings.ToUpper(algorithm) {
	case AlgorithmSHA1:
		return sha1.New, nil
	case AlgorithmSHA256:
		return sha256.New, nil
	case AlgorithmSHA512:
		return sha512.New, nil
	default:
		return nil, ErrInvalidAlgorithm
	}
}

// GenerateTOTPCode 基于密钥和当前时间生成TOTP代码
func GenerateTOTPCode(secret string, config TOTPConfig) (string, error) {
	// 获取哈希函数
	hashFunc, err := getHashFunc(config.Algorithm)
	if err != nil {
		return "", err
	}

	// 解码密钥
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", ErrInvalidSecret
	}

	// 获取当前时间步
	timeStep := uint64(time.Now().Unix() / int64(config.Period))

	// 生成TOTP代码
	return generateTOTP(secretBytes, timeStep, config, hashFunc)
}

// ValidateTOTPCode 验证TOTP代码
// 允许windowSize个时间周期的误差（默认前后1个）
func ValidateTOTPCode(secret, code string, config TOTPConfig, usedTokens *UsedTokenTracker) bool {
	// 获取哈希函数
	hashFunc, err := getHashFunc(config.Algorithm)
	if err != nil {
		return false
	}

	// 解码密钥
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}

	// 获取当前时间步
	timeStep := uint64(time.Now().Unix() / int64(config.Period))

	// 检查当前及前后windowSize个时间步的代码
	windowSize := config.WindowSize
	if windowSize <= 0 {
		windowSize = 1
	}

	for i := -windowSize; i <= windowSize; i++ {
		// 计算要检查的时间步
		checkStep := timeStep
		if i != 0 {
			if i < 0 && uint64(-i) > timeStep {
				// 避免时间步为负数
				continue
			}
			checkStep = timeStep + uint64(i)
		}

		// 生成验证代码
		validCode, err := generateTOTP(secretBytes, checkStep, config, hashFunc)
		if err != nil {
			continue
		}

		// 检查是否与提供的代码匹配
		if validCode == code {
			// 如果启用了令牌使用跟踪
			if config.SkipValidUsedTokens && usedTokens != nil {
				// 检查令牌是否已被使用
				tokenKey := fmt.Sprintf("%s:%d", secret, checkStep)
				if usedTokens.IsTokenUsed(tokenKey) {
					// 令牌已使用，拒绝验证
					continue
				}
				// 标记令牌为已使用
				usedTokens.MarkTokenAsUsed(tokenKey)
			}
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
func generateTOTP(secret []byte, timeStep uint64, config TOTPConfig, hashFunc func() hash.Hash) (string, error) {
	// 将时间步转换为字节数组
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, timeStep)

	// 计算HMAC
	h := hmac.New(hashFunc, secret)
	_, err := h.Write(msg)
	if err != nil {
		return "", err
	}

	// 获取哈希结果
	sum := h.Sum(nil)

	// 动态截断
	offset := sum[len(sum)-1] & 0xf
	binary := (uint32(sum[offset]&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])) % uint32(pow10(config.Digits))

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

// UsedTokenTracker 用于跟踪已使用的令牌，防止重放攻击
type UsedTokenTracker struct {
	usedTokens map[string]time.Time
	mutex      sync.RWMutex
	expiry     time.Duration
}

// NewUsedTokenTracker 创建新的令牌跟踪器
func NewUsedTokenTracker(expiry time.Duration) *UsedTokenTracker {
	if expiry <= 0 {
		// 默认保留24小时
		expiry = 24 * time.Hour
	}

	tracker := &UsedTokenTracker{
		usedTokens: make(map[string]time.Time),
		expiry:     expiry,
	}

	// 启动清理过期令牌的goroutine
	go tracker.cleanupExpiredTokens()

	return tracker
}

// MarkTokenAsUsed 标记令牌为已使用
func (t *UsedTokenTracker) MarkTokenAsUsed(tokenKey string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.usedTokens[tokenKey] = time.Now()
}

// IsTokenUsed 检查令牌是否已被使用
func (t *UsedTokenTracker) IsTokenUsed(tokenKey string) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	_, used := t.usedTokens[tokenKey]
	return used
}

// 清理过期的令牌
func (t *UsedTokenTracker) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		t.mutex.Lock()
		now := time.Now()
		for token, usedTime := range t.usedTokens {
			// 如果令牌过期，删除它
			if now.Sub(usedTime) > t.expiry {
				delete(t.usedTokens, token)
			}
		}
		t.mutex.Unlock()
	}
}

// TOTP 是一个简化使用的TOTP结构体
type TOTP struct {
	Secret      string
	Config      TOTPConfig
	BackupCodes []BackupCode
	UsedTokens  *UsedTokenTracker
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

	// 创建备份码
	backupCodes, err := GenerateBackupCodes(DefaultBackupCodeCount, DefaultBackupCodeLength)
	if err != nil {
		return nil, err
	}

	return &TOTP{
		Secret:      secret,
		Config:      cfg,
		BackupCodes: backupCodes,
		UsedTokens:  NewUsedTokenTracker(24 * time.Hour),
	}, nil
}

// NewTOTPWithSecret 使用已有密钥创建TOTP实例
func NewTOTPWithSecret(secret string, config ...TOTPConfig) (*TOTP, error) {
	cfg := DefaultTOTPConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// 验证密钥格式
	_, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return nil, ErrInvalidSecret
	}

	// 创建备份码
	backupCodes, err := GenerateBackupCodes(DefaultBackupCodeCount, DefaultBackupCodeLength)
	if err != nil {
		return nil, err
	}

	return &TOTP{
		Secret:      secret,
		Config:      cfg,
		BackupCodes: backupCodes,
		UsedTokens:  NewUsedTokenTracker(24 * time.Hour),
	}, nil
}

// Generate 生成当前TOTP代码
func (t *TOTP) Generate() (string, error) {
	return GenerateTOTPCode(t.Secret, t.Config)
}

// Validate 验证TOTP代码
func (t *TOTP) Validate(code string) bool {
	return ValidateTOTPCode(t.Secret, code, t.Config, t.UsedTokens)
}

// ValidateBackupCode 验证备份码
func (t *TOTP) ValidateBackupCode(code string) bool {
	for i, backupCode := range t.BackupCodes {
		if !backupCode.Used && backupCode.Code == code {
			// 标记为已使用
			t.BackupCodes[i].Used = true
			return true
		}
	}
	return false
}

// GetUnusedBackupCodes 获取未使用的备份码
func (t *TOTP) GetUnusedBackupCodes() []string {
	var codes []string
	for _, code := range t.BackupCodes {
		if !code.Used {
			codes = append(codes, code.Code)
		}
	}
	return codes
}

// RegenerateBackupCodes 重新生成备份码
func (t *TOTP) RegenerateBackupCodes() error {
	codes, err := GenerateBackupCodes(DefaultBackupCodeCount, DefaultBackupCodeLength)
	if err != nil {
		return err
	}
	t.BackupCodes = codes
	return nil
}

// UseBackupCode 使用一个备份码
func (t *TOTP) UseBackupCode(code string) error {
	for i, backupCode := range t.BackupCodes {
		if !backupCode.Used && backupCode.Code == code {
			t.BackupCodes[i].Used = true
			return nil
		}
	}

	// 检查是否有备份码找到但已使用
	for _, backupCode := range t.BackupCodes {
		if backupCode.Used && backupCode.Code == code {
			return ErrBackupCodeInvalid
		}
	}

	return ErrBackupCodeInvalid
}

// ProvisioningURI 生成配置URI
func (t *TOTP) ProvisioningURI(accountName string) string {
	return GenerateProvisioningURI(t.Secret, accountName, t.Config)
}

// GenerateBackupCodes 生成备份码
func GenerateBackupCodes(count, length int) ([]BackupCode, error) {
	if count <= 0 {
		count = DefaultBackupCodeCount
	}
	if length <= 0 {
		length = DefaultBackupCodeLength
	}

	codes := make([]BackupCode, count)
	chars := "0123456789"

	for i := 0; i < count; i++ {
		// 生成随机码
		codeBytes := make([]byte, length)
		_, err := rand.Read(codeBytes)
		if err != nil {
			return nil, err
		}

		// 将随机字节转换为数字字符
		var codeStr strings.Builder
		for _, b := range codeBytes {
			codeStr.WriteByte(chars[int(b)%len(chars)])
		}

		codes[i] = BackupCode{
			Code: codeStr.String(),
			Used: false,
		}
	}

	return codes, nil
}
