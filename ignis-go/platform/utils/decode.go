package utils

import (
	"encoding/base64"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func StringToBytesBase64HookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t != reflect.TypeOf([]byte{}) {
			return data, nil
		}

		return base64.StdEncoding.DecodeString(data.(string))
	}
}
