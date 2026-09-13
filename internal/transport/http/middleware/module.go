package middleware

import "go.uber.org/fx"

// Module bundles the middleware package's constructors as fx providers.
var Module = fx.Module("middleware",
	fx.Provide(fx.Annotate(
		AuthMiddleware,
		fx.ResultTags(`name:"authMiddleware"`),
	)),
)
