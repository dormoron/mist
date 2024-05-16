package utils

import (
	"fmt"
)

type AnyValue struct {
	Val any
	Err error
}

func (av AnyValue) String() (string, error) {
	if av.Err != nil {
		return "", av.Err
	}
	val, ok := av.Val.(string)
	if !ok {
		return "", fmt.Errorf("type conversion failed, expected type:%s, actual value:%#v", "string", av.Val)
	}
	return val, nil
}

func (av AnyValue) StringOrDefault(def string) string {
	val, err := av.String()
	if err != nil {
		return def
	}
	return val
}
