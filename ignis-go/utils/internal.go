package utils

import (
	"encoding/base64"
	"reflect"

	"github.com/9triver/ignis/utils/errors"
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

func StructToMap(input any) (m map[string]any, err error) {
	m = make(map[string]any)
	t := reflect.TypeOf(input)
	v := reflect.ValueOf(input)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		v = v.Elem()
	}

	if t.Kind() != reflect.Struct {
		err = errors.New("input type is not a struct")
		return
	}

	for i := range t.NumField() {
		field := v.Field(i)
		name := t.Field(i).Name
		m[name] = field.Interface()
	}
	return
}

type MapHook func(dstType, srcType reflect.Type, v any) (any, error)

var (
	base64Hook MapHook = func(dstType, srcType reflect.Type, v any) (any, error) {
		if dstType == reflect.TypeFor[[]byte]() && srcType.Kind() == reflect.String {
			decoded, err := base64.StdEncoding.DecodeString(v.(string))
			if err != nil {
				return nil, err
			}
			return decoded, nil
		}
		return nil, nil
	}

	chanConvertHook MapHook = func(dstType, srcType reflect.Type, v any) (any, error) {
		if dstType.Kind() == reflect.Chan && srcType.Kind() == reflect.Chan {
			if dstType.Elem() == srcType.Elem() {
				return v, nil
			}

			vch := reflect.ValueOf(v)
			ch := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, dstType.Elem()), 10)
			go func() {
				defer ch.Close()
				for {
					elem, ok := vch.Recv()
					if !ok {
						return
					}
					// TODO: make workaround more generic
					var value reflect.Value
					if o, ok := elem.Interface().(interface {
						GetValue() (any, error) // cannot import Object interface due to circular dependencies
					}); ok {
						v, err := o.GetValue()
						if err != nil {
							return
						}
						value = reflect.ValueOf(v)
					} else {
						value = elem.Elem()
					}
					ch.Send(value.Convert(dstType.Elem()))
				}
			}()
			return ch.Interface(), nil
		}

		return nil, nil
	}

	hooks = []MapHook{base64Hook, chanConvertHook}
)

func MapToStruct[I any](invoke map[string]any) (ret I, err error) {
	var input I

	t := reflect.TypeFor[I]()
	if t.Kind() == reflect.Pointer {
		input = reflect.New(t.Elem()).Interface().(I)
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		err = errors.New("output type is not a struct")
		return
	}

	v := reflect.ValueOf(&input)
	for i := range t.NumField() {
		fieldValue := v.Elem().Field(i)
		field := t.Field(i)

		if v, ok := invoke[field.Name]; ok {
			valueType := reflect.TypeOf(v)

			for _, hook := range hooks {
				converted, err := hook(field.Type, valueType, v)
				if err != nil {
					return ret, err
				}
				if converted != nil {
					v = converted
					break
				}
			}

			fieldValue.Set(reflect.ValueOf(v).Convert(field.Type))
		}
	}

	return input, nil
}
