package executors

import (
	"errors"

	"go.uber.org/fx"

	"github.com/DoomSentinel/scheduler/scheduler/types"
)

var Module = fx.Provide(
	NewRemoteHTTPClient,
	fx.Annotated{
		Group:  types.ExecutorsGroup,
		Target: NewRemoteHttp,
	},
	fx.Annotated{
		Group:  types.ExecutorsGroup,
		Target: NewDummy,
	},
	fx.Annotated{
		Group:  types.ExecutorsGroup,
		Target: NewCommand,
	},
)

var (
	ErrUnexpectedStatusCode = errors.New("received unexpected status code")
)
