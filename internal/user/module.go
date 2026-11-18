package user

import "go.uber.org/fx"

// Module bundles the user package's constructors as fx providers.
var Module = fx.Module("user",
	fx.Provide(fx.Annotate(NewPostgresRepository, fx.As(new(Repository)))),
	// fx.Provide(NewService),
	// fx.Provide(NewHandler),
)
