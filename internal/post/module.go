package post

import "go.uber.org/fx"

// Module bundles the post package's constructors as fx providers.
var Module = fx.Module("post",
	fx.Provide(NewPostgresRepository),
)
