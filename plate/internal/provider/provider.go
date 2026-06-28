package provider

import (
	"context"

	"github.com/aamoghS/sideprojects/plate/internal/plans"
	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

type Backend interface {
	Name() string
	Create(ctx context.Context, inst vm.Instance, plan plans.Plan) (backendID string, err error)
	Start(ctx context.Context, inst vm.Instance) error
	Stop(ctx context.Context, inst vm.Instance) error
	Delete(ctx context.Context, inst vm.Instance) error
	Sync(ctx context.Context, inst vm.Instance) (vm.Instance, error)
}
