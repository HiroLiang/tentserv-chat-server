package config

import (
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/spf13/viper"
)

var ()

// AppConfig - struct of app config.
// Add more config here if any new config been added in config.yaml.
type AppConfig struct {
	Database map[string]*DBConfig `mapstructure:"databases"`
	Redis    RedisConfig          `mapstructure:"redis"`

	AuthToken struct {
		Expiration time.Duration `mapstructure:"expiration"`
	} `mapstructure:"auth_token"`

	Secrets struct {
		HmacSecret string `mapstructure:"hmac"`
	} `mapstructure:"secrets"`

	RateLimitConfig struct {
		GlobalLimit int64         `mapstructure:"global_limit"`
		IPLimit     int64         `mapstructure:"ip_limit"`
		UserLimit   int64         `mapstructure:"user_limit"`
		GlobalUnit  time.Duration `mapstructure:"global_unit"`
		IPUnit      time.Duration `mapstructure:"ip_unit"`
		UserUnit    time.Duration `mapstructure:"user_unit"`
	} `mapstructure:"rate_limit_config"`

	Storage struct {
		BasePath string `mapstructure:"base_path"`
		BaseURL  string `mapstructure:"base_url"`
	} `mapstructure:"storage"`

	Email struct {
		SenderAddress string        `mapstructure:"sender_address"`
		SenderName    string        `mapstructure:"sender_name"`
		ApiKey        string        `mapstructure:"api_key"`
		BaseURL       string        `mapstructure:"base_url"`
		VerifyTTL     time.Duration `mapstructure:"verify_ttl"`
	} `mapstructure:"email"`
}

type DBConfig struct {
	Driver string        `mapstructure:"driver"`
	Dsn    string        `mapstructure:"dsn"`
	Pool   *DBPoolConfig `mapstructure:"pool"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type DBPoolConfig struct {
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// global singleton config
var (
	reBraceEnv    = regexp.MustCompile(`\$\{[^}]+}`)
	reBraceEnvExp = regexp.MustCompile(`\$\{([^:}]+)(?::([^}]+))?}`)

	config     *AppConfig
	configOnce sync.Once
	configErr  error
)

// App - safe accessor that lazy load default config if not loaded yet
func App() *AppConfig {
	if config == nil {
		panic("config not loaded, call LoadConfig first")
	}
	return config
}

// LoadConfig - Load config YAML in deliver path.
// path: config folder path
func LoadConfig(path string) error {
	configOnce.Do(func() {
		configErr = load(path)
	})

	return configErr
}

func load(path string) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config error: %w", err)
	}

	for _, key := range viper.AllKeys() {
		val := viper.GetString(key)
		if containsBraceEnv(val) {
			val = expandEnvWithDefault(val)
		} else {
			val = os.ExpandEnv(val)
		}
		viper.Set(key, val)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("unmarshal config error: %w", err)
	}

	return nil
}

func expandEnvWithDefault(s string) string {
	return reBraceEnvExp.ReplaceAllStringFunc(s, func(sub string) string {
		matches := reBraceEnvExp.FindStringSubmatch(sub)
		if len(matches) >= 2 {
			key := matches[1]
			def := ""
			if len(matches) >= 3 {
				def = matches[2]
			}
			if val, ok := os.LookupEnv(key); ok && val != "" {
				return val
			}
			return def
		}
		return sub
	})
}

func containsBraceEnv(s string) bool {
	return reBraceEnv.MatchString(s)
}
