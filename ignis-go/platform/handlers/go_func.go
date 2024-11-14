package handlers

import (
	"actors/platform/store"
	"actors/platform/utils"
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type GoFunction[T, R any] struct {
	FuncDec
	impl     func(req *T) (R, error)
	language store.Language
}

func ImplGo[T, R any](
	def FuncDec,
	handler func(req *T) (R, error),
	language store.Language,
) *GoFunction[T, R] {
	return &GoFunction[T, R]{FuncDec: def, impl: handler, language: language}
}

func NewGo[T, R any](
	name string,
	handler func(req *T) (R, error),
	language store.Language,
) *GoFunction[T, R] {
	return &GoFunction[T, R]{FuncDec: DeclareTyped[T, R](name), impl: handler, language: language}
}

func (h *GoFunction[T, R]) Call(params map[string]*store.Object, s *store.Store) (*store.Object, error) {
	realParams := make(map[string]any)
	var req T

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: utils.StringToBytesBase64HookFunc(),
		Result:     &req,
	})
	if err != nil {
		return nil, err
	}

	for k, v := range params {
		switch v.Language {
		case store.LangGo, store.LangJson:
			realParams[k] = v.Value
		default:
			return nil, fmt.Errorf("param %s: unsupported language %s", k, v.Language)
		}
	}

	if err := decoder.Decode(realParams); err != nil {
		return nil, err
	}

	obj, err := h.impl(&req)
	if err != nil {
		return nil, err
	}
	return s.Add(obj, h.Language()), nil
}

func (h *GoFunction[T, R]) Language() store.Language {
	return h.language
}
