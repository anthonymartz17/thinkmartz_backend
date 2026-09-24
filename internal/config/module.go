package config

import (
	"go.uber.org/fx"
)

// Module bundles the config package's constructors as fx providers.
var Module = fx.Module("config",

	fx.Provide(Load),
	fx.Provide(NewAppConfig),
	fx.Provide(NewDBConfig),
	fx.Provide(NewRedisConfig),
	fx.Provide(NewJWTConfig),
	fx.Provide(ProvideCelebrityThreshold),
)
