package utils

import (
	"encoding/base64"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func fieldsOf[T any]() (fields []string) {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}
	for idx := range t.NumField() {
		fields = append(fields, t.Field(idx).Name)
	}
	return
}

// TODO: use BSON to prevent using this ugly hook for binary data
func stringToBytesBase64HookFunc() mapstructure.DecodeHookFunc {
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

func MapToStruct[I any](invoke map[string]any) (ret I, err error) {
	var input I
	var dec any

	t := reflect.TypeFor[I]()
	if t.Kind() == reflect.Pointer {
		input = reflect.New(t.Elem()).Interface().(I)
		dec = input
	} else {
		dec = &input
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: stringToBytesBase64HookFunc(),
		Result:     dec,
	})
	if err != nil {
		return
	}
	if err = decoder.Decode(invoke); err != nil {
		return
	}
	return input, nil
}
