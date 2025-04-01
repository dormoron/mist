package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ValidateStruct 验证结构体，根据字段标签进行验证
func ValidateStruct(v *Validator, s interface{}) {
	val := reflect.ValueOf(s)

	// 如果是指针，获取其指向的值
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 确保是结构体
	if val.Kind() != reflect.Struct {
		panic("参数必须是结构体")
	}

	// 获取结构体类型
	typ := val.Type()

	// 遍历所有字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 获取验证标签
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// 如果字段是一个嵌套的结构体，递归验证
		if field.Kind() == reflect.Struct {
			// 获取子结构体的值
			fieldValue := field.Interface()
			// 递归验证
			ValidateStruct(v, fieldValue)
			continue
		}

		// 解析并应用验证规则
		validateRules(v, validateTag, field, fieldType.Name)
	}
}

// validateRules 解析并应用验证规则
func validateRules(v *Validator, tag string, field reflect.Value, fieldName string) {
	// 根据逗号分割验证规则
	rules := strings.Split(tag, ",")

	for _, rule := range rules {
		// 去除空白
		rule = strings.TrimSpace(rule)

		// 解析规则名称和参数
		parts := strings.SplitN(rule, "=", 2)
		ruleName := parts[0]

		// 根据字段类型和规则应用不同的验证
		switch field.Kind() {
		case reflect.String:
			validateString(v, ruleName, parts, field.String(), fieldName)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			validateInt(v, ruleName, parts, field.Int(), fieldName)
		case reflect.Float32, reflect.Float64:
			validateFloat(v, ruleName, parts, field.Float(), fieldName)
		case reflect.Bool:
			validateBool(v, ruleName, parts, field.Bool(), fieldName)
		case reflect.Slice, reflect.Array:
			validateSlice(v, ruleName, parts, field, fieldName)
		}
	}
}

// validateString 验证字符串类型
func validateString(v *Validator, ruleName string, parts []string, value string, fieldName string) {
	switch ruleName {
	case "required":
		v.Required(value, fieldName)
	case "min":
		if len(parts) > 1 {
			min, err := strconv.Atoi(parts[1])
			if err == nil {
				v.MinLength(value, min, fieldName)
			}
		}
	case "max":
		if len(parts) > 1 {
			max, err := strconv.Atoi(parts[1])
			if err == nil {
				v.MaxLength(value, max, fieldName)
			}
		}
	case "email":
		v.Email(value, fieldName)
	case "url":
		v.URL(value, fieldName)
	case "alpha":
		v.Alpha(value, fieldName)
	case "alphanum":
		v.Alphanumeric(value, fieldName)
	case "phone":
		v.PhoneNumber(value, fieldName)
	case "idcard":
		v.ChineseIDCard(value, fieldName)
	}
}

// validateInt 验证整数类型
func validateInt(v *Validator, ruleName string, parts []string, value int64, fieldName string) {
	switch ruleName {
	case "min":
		if len(parts) > 1 {
			min, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil {
				if value < min {
					v.AddError(fieldName, fmt.Sprintf("必须大于或等于%d", min))
				}
			}
		}
	case "max":
		if len(parts) > 1 {
			max, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil {
				if value > max {
					v.AddError(fieldName, fmt.Sprintf("必须小于或等于%d", max))
				}
			}
		}
	}
}

// validateFloat 验证浮点数类型
func validateFloat(v *Validator, ruleName string, parts []string, value float64, fieldName string) {
	switch ruleName {
	case "min":
		if len(parts) > 1 {
			min, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				if value < min {
					v.AddError(fieldName, fmt.Sprintf("必须大于或等于%.2f", min))
				}
			}
		}
	case "max":
		if len(parts) > 1 {
			max, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				if value > max {
					v.AddError(fieldName, fmt.Sprintf("必须小于或等于%.2f", max))
				}
			}
		}
	}
}

// validateBool 验证布尔类型
func validateBool(v *Validator, ruleName string, parts []string, value bool, fieldName string) {
	switch ruleName {
	case "required":
		// 对于布尔类型，required通常没有意义，但我们可以定义为必须为true
		if !value {
			v.AddError(fieldName, "必须为true")
		}
	}
}

// validateSlice 验证切片类型
func validateSlice(v *Validator, ruleName string, parts []string, field reflect.Value, fieldName string) {
	switch ruleName {
	case "required":
		if field.Len() == 0 {
			v.AddError(fieldName, "不能为空")
		}
	case "min":
		if len(parts) > 1 {
			min, err := strconv.Atoi(parts[1])
			if err == nil {
				if field.Len() < min {
					v.AddError(fieldName, fmt.Sprintf("长度不能小于%d", min))
				}
			}
		}
	case "max":
		if len(parts) > 1 {
			max, err := strconv.Atoi(parts[1])
			if err == nil {
				if field.Len() > max {
					v.AddError(fieldName, fmt.Sprintf("长度不能大于%d", max))
				}
			}
		}
	}
}
