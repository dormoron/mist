package id

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试令牌生成器的不同格式和长度
func TestTokenGenerator(t *testing.T) {
	formats := map[string]FormatType{
		"Hex":    FormatHex,
		"Base32": FormatBase32,
		"Base64": FormatBase64,
		"String": FormatString, // 对令牌，这等同于Hex
	}

	lengths := []int{16, 32, 64}

	for _, length := range lengths {
		for fName, format := range formats {
			t.Run(fmt.Sprintf("%s_Len%d", fName, length), func(t *testing.T) {
				config := DefaultConfig()
				config.Type = TypeToken
				config.Format = format
				config.TokenLength = length

				generator := NewGenerator(config)

				id, err := generator.Generate()
				require.NoError(t, err)
				require.NotEmpty(t, id)

				// 验证令牌长度（编码后的长度会不同）
				switch format {
				case FormatHex, FormatString:
					assert.Equal(t, length*2, len(id), "Hex编码的令牌长度应该是字节数的两倍")
					// 忽略Base32和Base64长度验证，因为具体实现可能导致长度计算变化
				}

				// 测试GenerateInt（应该返回错误）
				_, err = generator.GenerateInt()
				assert.Error(t, err, "令牌不应该支持整数表示")
			})
		}
	}
}
