//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package catalog

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// handler is one trigger in this domain's own internal sequence — see
// training's identical contract.go comment for why it's declared here
// rather than imported from the parent chain package.
type handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// catalogList shows the "📚 Каталог" screen (catalog:list and its
// pagination/city-filter callbacks).
type catalogList interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// catalogInfo shows a single catalog entry's card (catalog:view:{id}).
type catalogInfo interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// catalogAdd owns "➕ В мои" (catalog:add:{id}).
type catalogAdd interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}
