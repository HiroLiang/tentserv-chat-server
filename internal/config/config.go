package config

import (
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// AppConfig - struct of app config.
// Add more config here if any new config been added in config.yaml.
type AppConfig struct {
	AuthToken struct {
		Expiration time.Duration `mapstructure:"expiration"`
	} `mapstructure:"auth_token"`

	Secrets struct {
		HmacSecret string `mapstructure:"HMAC_SECRET"`
	} `mapstructure:"secrets"`

	RateLimitConfig struct {
		GlobalLimit int           `mapstructure:"global_limit"`
		GlobalUnit  time.Duration `mapstructure:"global_unit"`
		IPLimit     int           `mapstructure:"ip_limit"`
		IPUnit      time.Duration `mapstructure:"ip_unit"`
		UserLimit   int           `mapstructure:"user_limit"`
		UserUnit    time.Duration `mapstructure:"user_unit"`
	} `mapstructure:"rate_limit_config"`

	Database map[string]*DBConfig `mapstructure:"databases"`

	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`

	Storage struct {
		LocalPath string `mapstructure:"local_path"`
		BaseURL   string `mapstructure:"base_url"`
	} `mapstructure:"storage"`
}

type DBPoolConfig struct {
	MaxOpenConns    int `mapstructure:"max_open_conns"`
	MaxIdleConns    int `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime int `mapstructure:"conn_max_idle_time"`
}

type DBConfig struct {
	Driver string        `mapstructure:"driver"`
	Dsn    string        `mapstructure:"dsn"`
	Pool   *DBPoolConfig `mapstructure:"pool"`
}

// global singleton config
var (
	config     *AppConfig
	configOnce sync.Once
	configErr  error
)

// App - safe accessor that lazy load default config if not loaded yet
func App() *AppConfig {
	configOnce.Do(func() {
		configErr = load("./config")
	})

	if configErr != nil {
		panic(fmt.Sprintf("config not loaded: %v", configErr))
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

	fmt.Printf("Config loaded \n")
	return nil
}

func expandEnvWithDefault(s string) string {
	re := regexp.MustCompile(`\$\{([^:}]+)(?::([^}]+))?}`)
	return re.ReplaceAllStringFunc(s, func(sub string) string {
		matches := re.FindStringSubmatch(sub)
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
	return regexp.MustCompile(`\$\{[^}]+}`).MatchString(s)
}
