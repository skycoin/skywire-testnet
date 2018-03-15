package config

import (
	"math/big"
	"time"

	"github.com/go-akka/configuration"
)

type HOCONConfiguration struct {
	*configuration.Config
}

func NewHOCONConfiguration(conf *configuration.Config) Configuration {
	return &HOCONConfiguration{
		conf,
	}
}

func (p *HOCONConfiguration) GetConfig(path string) Configuration {
	if p == nil || p.Config == nil {
		return (*HOCONConfiguration)(nil)
	}

	conf := p.Config.GetConfig(path)
	if conf == nil {
		return (*HOCONConfiguration)(nil)
	}

	return &HOCONConfiguration{conf}
}

func (p *HOCONConfiguration) WithFallback(fallback Configuration) Configuration {
	if fallback == nil {
		return p
	}

	p.Config = p.Config.WithFallback(fallback.(*HOCONConfiguration).Config)
	return p
}

func (p *HOCONConfiguration) GetBoolean(path string, defaultVal ...bool) bool {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return false
	}
	return p.Config.GetBoolean(path, defaultVal...)
}

func (p *HOCONConfiguration) GetByteSize(path string) *big.Int {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetByteSize(path)
}

func (p *HOCONConfiguration) GetInt32(path string, defaultVal ...int32) int32 {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return p.Config.GetInt32(path, defaultVal...)
}

func (p *HOCONConfiguration) GetInt64(path string, defaultVal ...int64) int64 {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return p.Config.GetInt64(path, defaultVal...)
}

func (p *HOCONConfiguration) GetString(path string, defaultVal ...string) string {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return ""
	}
	return p.Config.GetString(path, defaultVal...)
}

func (p *HOCONConfiguration) GetFloat32(path string, defaultVal ...float32) float32 {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return p.Config.GetFloat32(path, defaultVal...)
}

func (p *HOCONConfiguration) GetFloat64(path string, defaultVal ...float64) float64 {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return p.Config.GetFloat64(path, defaultVal...)
}

func (p *HOCONConfiguration) GetTimeDuration(path string, defaultVal ...time.Duration) time.Duration {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return p.Config.GetTimeDuration(path, defaultVal...)
}

func (p *HOCONConfiguration) GetTimeDurationInfiniteNotAllowed(path string, defaultVal ...time.Duration) time.Duration {
	if p == nil || p.Config == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return p.Config.GetTimeDurationInfiniteNotAllowed(path, defaultVal...)
}

func (p *HOCONConfiguration) GetBooleanList(path string) []bool {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetBooleanList(path)
}

func (p *HOCONConfiguration) GetFloat32List(path string) []float32 {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetFloat32List(path)
}

func (p *HOCONConfiguration) GetFloat64List(path string) []float64 {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetFloat64List(path)
}

func (p *HOCONConfiguration) GetInt32List(path string) []int32 {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetInt32List(path)
}

func (p *HOCONConfiguration) GetInt64List(path string) []int64 {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetInt64List(path)
}

func (p *HOCONConfiguration) GetByteList(path string) []byte {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetByteList(path)
}

func (p *HOCONConfiguration) GetStringList(path string) []string {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.GetStringList(path)
}

func (p *HOCONConfiguration) HasPath(path string) bool {
	if p == nil || p.Config == nil {
		return false
	}
	return p.Config.HasPath(path)
}

func (p *HOCONConfiguration) Keys() []string {
	if p == nil || p.Config == nil {
		return nil
	}
	return p.Config.Root().GetObject().GetKeys()
}

func (p *HOCONConfiguration) IsEmpty() bool {
	return p == nil || p.Config.IsEmpty()
}

func (p *HOCONConfiguration) String() string {
	if p == nil || p.Config.IsEmpty() {
		return ""
	}

	return p.Config.String()
}

type HOCONConfigProvider struct {
}

func (p *HOCONConfigProvider) LoadConfig(filename string) Configuration {
	conf := configuration.LoadConfig(filename)
	return NewHOCONConfiguration(conf)
}

func (p *HOCONConfigProvider) ParseString(cfgStr string) Configuration {
	conf := configuration.ParseString(cfgStr)
	return NewHOCONConfiguration(conf)
}
