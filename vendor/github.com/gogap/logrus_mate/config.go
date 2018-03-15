package logrus_mate

import (
	"github.com/gogap/config"
)

type Option func(*Config)

type Config struct {
	ConfigFile   string
	ConfigString string

	configOpts []config.Option
}

func ConfigFile(fn string) Option {
	return func(o *Config) {
		o.configOpts = append(o.configOpts, config.ConfigFile(fn))
	}
}

func ConfigString(str string) Option {
	return func(o *Config) {
		o.configOpts = append(o.configOpts, config.ConfigString(str))
	}
}

func ConfigProvider(provider config.ConfigurationProvider) Option {
	return func(o *Config) {
		o.configOpts = append(o.configOpts, config.ConfigProvider(provider))
	}
}
