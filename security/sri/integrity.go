package sri

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// 支持的哈希算法
const (
	AlgoSha256 = "sha256"
	AlgoSha384 = "sha384"
	AlgoSha512 = "sha512"
)

// 错误定义
var (
	ErrUnsupportedAlgorithm = errors.New("不支持的哈希算法")
	ErrInvalidIntegrity     = errors.New("无效的完整性字符串")
	ErrResourceNotFound     = errors.New("资源未找到")
	ErrReadFailed           = errors.New("读取资源失败")
)

// IntegrityHash 表示完整性哈希值
type IntegrityHash struct {
	Algorithm string // 哈希算法
	Hash      string // Base64编码的哈希值
}

// String 返回完整的完整性字符串，如 "sha256-..."
func (ih IntegrityHash) String() string {
	return fmt.Sprintf("%s-%s", ih.Algorithm, ih.Hash)
}

// Verify 验证内容与完整性哈希是否匹配
func (ih IntegrityHash) Verify(content []byte) bool {
	hash, err := CalculateHash(content, ih.Algorithm)
	if err != nil {
		return false
	}
	return hash.Hash == ih.Hash
}

// ParseIntegrity 解析完整性字符串，如 "sha256-..."
func ParseIntegrity(integrity string) (IntegrityHash, error) {
	parts := strings.SplitN(integrity, "-", 2)
	if len(parts) != 2 {
		return IntegrityHash{}, ErrInvalidIntegrity
	}

	algo := parts[0]
	hash := parts[1]

	// 验证算法
	switch algo {
	case AlgoSha256, AlgoSha384, AlgoSha512:
		// 支持的算法
	default:
		return IntegrityHash{}, ErrUnsupportedAlgorithm
	}

	// 验证哈希是否是有效的Base64
	_, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return IntegrityHash{}, ErrInvalidIntegrity
	}

	return IntegrityHash{
		Algorithm: algo,
		Hash:      hash,
	}, nil
}

// CalculateHash 计算内容的哈希值
func CalculateHash(content []byte, algorithm string) (IntegrityHash, error) {
	var h hash.Hash

	switch algorithm {
	case AlgoSha256:
		h = sha256.New()
	case AlgoSha384:
		h = sha512.New384()
	case AlgoSha512:
		h = sha512.New()
	default:
		return IntegrityHash{}, ErrUnsupportedAlgorithm
	}

	h.Write(content)
	hashBytes := h.Sum(nil)
	hashBase64 := base64.StdEncoding.EncodeToString(hashBytes)

	return IntegrityHash{
		Algorithm: algorithm,
		Hash:      hashBase64,
	}, nil
}

// CalculateFileHash 计算文件的哈希值
func CalculateFileHash(filePath string, algorithm string) (IntegrityHash, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return IntegrityHash{}, fmt.Errorf("%w: %v", ErrResourceNotFound, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return IntegrityHash{}, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	return CalculateHash(content, algorithm)
}

// CalculateURLHash 计算URL资源的哈希值
func CalculateURLHash(url string, algorithm string) (IntegrityHash, error) {
	resp, err := http.Get(url)
	if err != nil {
		return IntegrityHash{}, fmt.Errorf("%w: %v", ErrResourceNotFound, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return IntegrityHash{}, fmt.Errorf("%w: HTTP %d", ErrResourceNotFound, resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return IntegrityHash{}, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	return CalculateHash(content, algorithm)
}

// IsValidAlgorithm 检查算法是否有效
func IsValidAlgorithm(algorithm string) bool {
	switch algorithm {
	case AlgoSha256, AlgoSha384, AlgoSha512:
		return true
	default:
		return false
	}
}

// GenerateIntegrityTag 为HTML元素生成完整性标签属性
func GenerateIntegrityTag(resourcePath, algorithm string) (string, error) {
	var hash IntegrityHash
	var err error

	// 检查算法
	if !IsValidAlgorithm(algorithm) {
		return "", ErrUnsupportedAlgorithm
	}

	// 判断是URL还是本地文件
	if strings.HasPrefix(resourcePath, "http://") || strings.HasPrefix(resourcePath, "https://") {
		hash, err = CalculateURLHash(resourcePath, algorithm)
	} else {
		hash, err = CalculateFileHash(resourcePath, algorithm)
	}

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("integrity=\"%s\"", hash.String()), nil
}

// BatchGenerateIntegrityTags 批量生成完整性标签
func BatchGenerateIntegrityTags(directory, ext, algorithm string) (map[string]string, error) {
	results := make(map[string]string)

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理指定扩展名的文件
		if !info.IsDir() && strings.HasSuffix(path, ext) {
			hash, err := CalculateFileHash(path, algorithm)
			if err != nil {
				return err
			}

			// 获取相对路径作为键
			relPath, err := filepath.Rel(directory, path)
			if err != nil {
				relPath = path // 回退到使用完整路径
			}

			results[relPath] = hash.String()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// VerifyResourceIntegrity 验证资源的完整性
func VerifyResourceIntegrity(resourcePath, integrityString string) (bool, error) {
	hash, err := ParseIntegrity(integrityString)
	if err != nil {
		return false, err
	}

	var content []byte

	// 判断是URL还是本地文件
	if strings.HasPrefix(resourcePath, "http://") || strings.HasPrefix(resourcePath, "https://") {
		resp, err := http.Get(resourcePath)
		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrResourceNotFound, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("%w: HTTP %d", ErrResourceNotFound, resp.StatusCode)
		}

		content, err = io.ReadAll(resp.Body)
		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrReadFailed, err)
		}
	} else {
		file, err := os.Open(resourcePath)
		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrResourceNotFound, err)
		}
		defer file.Close()

		content, err = io.ReadAll(file)
		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrReadFailed, err)
		}
	}

	return hash.Verify(content), nil
}
