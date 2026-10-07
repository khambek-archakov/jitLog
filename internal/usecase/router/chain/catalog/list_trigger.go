package catalog

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackList       = "catalog:list"
	callbackPagePrefix = "catalog:list:page:"
	callbackCityPrompt = "catalog:city:prompt"
	callbackCityAll    = "catalog:city:all"
)

// listTrigger covers the "📚 Каталог" screen's own entry callback, its
// pagination and its city-filter controls — all driven by
// catalog/list.Handle.
type listTrigger struct {
	list catalogList
}

func (h *listTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return model.ErrSkip
	}

	matches := in.CallbackData == callbackList ||
		strings.HasPrefix(in.CallbackData, callbackPagePrefix) ||
		in.CallbackData == callbackCityPrompt ||
		in.CallbackData == callbackCityAll

	if !matches {
		return model.ErrSkip
	}

	return h.list.Handle(ctx, u, in)
}
