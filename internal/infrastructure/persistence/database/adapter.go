package database

import (
	"github.com/HiroLiang/goat-server/internal/config"
)

func BuildDatabaseConfigs(cfg map[string]*config.DBConfig) map[string]ConnectionConfig {

	result := make(map[string]ConnectionConfig)

	for name, c := range cfg {
		cc := ConnectionConfig{
			Driver: c.Driver,
			Dsn:    c.Dsn,
		}

		if c.Pool != nil {
			cc.MaxOpenConns = c.Pool.MaxOpenConns
			cc.MaxIdleConns = c.Pool.MaxIdleConns
			cc.ConnMaxLifetime = c.Pool.ConnMaxLifetime
			cc.ConnMaxIdleTime = c.Pool.ConnMaxIdleTime
		}

		result[name] = cc
	}

	return result
}
