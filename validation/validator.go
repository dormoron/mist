package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ValidationError 表示验证错误
type ValidationError struct {
	Field   string // 字段名称
	Message string // 错误信息
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validator 提供数据验证功能
type Validator struct {
	Errors []ValidationError
}

// NewValidator 创建一个新的验证器
func NewValidator() *Validator {
	return &Validator{
		Errors: make([]ValidationError, 0),
	}
}

// Valid 检查验证器是否有错误
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError 添加一个验证错误
func (v *Validator) AddError(field, message string) {
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// Check 检查条件是否成立，如果不成立则添加错误
func (v *Validator) Check(ok bool, field, message string) {
	if !ok {
		v.AddError(field, message)
	}
}

// Required 检查字符串是否非空
func (v *Validator) Required(value string, field string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "不能为空")
	}
}

// MinLength 检查字符串最小长度
func (v *Validator) MinLength(value string, min int, field string) {
	if utf8.RuneCountInString(value) < min {
		v.AddError(field, fmt.Sprintf("长度不能小于%d个字符", min))
	}
}

// MaxLength 检查字符串最大长度
func (v *Validator) MaxLength(value string, max int, field string) {
	if utf8.RuneCountInString(value) > max {
		v.AddError(field, fmt.Sprintf("长度不能大于%d个字符", max))
	}
}

// Between 检查字符串长度是否在指定范围内
func (v *Validator) Between(value string, min, max int, field string) {
	length := utf8.RuneCountInString(value)
	if length < min || length > max {
		v.AddError(field, fmt.Sprintf("长度必须在%d到%d个字符之间", min, max))
	}
}

// Email 验证邮箱格式
func (v *Validator) Email(value string, field string) {
	if value == "" {
		return
	}

	_, err := mail.ParseAddress(value)
	if err != nil {
		v.AddError(field, "邮箱格式不正确")
	}
}

// URL 验证URL格式
func (v *Validator) URL(value string, field string) {
	if value == "" {
		return
	}

	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		v.AddError(field, "URL格式不正确")
	}
}

// Alpha 验证字符串仅包含字母
func (v *Validator) Alpha(value string, field string) {
	if value == "" {
		return
	}

	matched, _ := regexp.MatchString("^[a-zA-Z]+$", value)
	if !matched {
		v.AddError(field, "只能包含字母")
	}
}

// Alphanumeric 验证字符串仅包含字母和数字
func (v *Validator) Alphanumeric(value string, field string) {
	if value == "" {
		return
	}

	matched, _ := regexp.MatchString("^[a-zA-Z0-9]+$", value)
	if !matched {
		v.AddError(field, "只能包含字母和数字")
	}
}

// Numeric 验证字符串是否为数字
func (v *Validator) Numeric(value string, field string) {
	if value == "" {
		return
	}

	_, err := strconv.ParseFloat(value, 64)
	if err != nil {
		v.AddError(field, "必须是数字")
	}
}

// Integer 验证字符串是否为整数
func (v *Validator) Integer(value string, field string) {
	if value == "" {
		return
	}

	_, err := strconv.Atoi(value)
	if err != nil {
		v.AddError(field, "必须是整数")
	}
}

// Range 验证数字是否在指定范围内
func (v *Validator) Range(value int, min, max int, field string) {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("必须在%d到%d之间", min, max))
	}
}

// RangeFloat 验证浮点数是否在指定范围内
func (v *Validator) RangeFloat(value float64, min, max float64, field string) {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("必须在%.2f到%.2f之间", min, max))
	}
}

// PhoneNumber 验证手机号格式(中国大陆)
func (v *Validator) PhoneNumber(value string, field string) {
	if value == "" {
		return
	}

	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, value)
	if !matched {
		v.AddError(field, "手机号格式不正确")
	}
}

// ChineseIDCard 验证中国大陆身份证号
func (v *Validator) ChineseIDCard(value string, field string) {
	if value == "" {
		return
	}

	// 18位身份证正则
	matched, _ := regexp.MatchString(`^\d{17}[\dXx]$`, value)
	if !matched {
		v.AddError(field, "身份证号格式不正确")
		return
	}

	// 可以添加更复杂的验证，如校验码验证等
}

// InList 验证值是否在列表中
func (v *Validator) InList(value string, list []string, field string) {
	for _, item := range list {
		if value == item {
			return
		}
	}
	v.AddError(field, "值不在允许的范围内")
}

// NotInList 验证值是否不在列表中
func (v *Validator) NotInList(value string, list []string, field string) {
	for _, item := range list {
		if value == item {
			v.AddError(field, "值不允许使用")
			return
		}
	}
}

// Custom 自定义验证
func (v *Validator) Custom(ok bool, field, message string) {
	if !ok {
		v.AddError(field, message)
	}
}
