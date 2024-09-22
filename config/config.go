package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

type Config struct {
	GrpcMaxIdleSec  int     `env:"GRPC_MAX_IDLE_SEC" default:"3600"`
	SpaceWeaveAddr  string  `env:"SPACE_WEAVE_ADDR" default:":22500"`
	UnitSize        uint64  `env:"UNIT_SIZE" default:"4096"`
	TotalSize       uint64  `env:"TOTAL_SIZE" default:"1099511627776"` // 1 TiB
	SmallBlockRatio float64 `env:"SMALL_BLOCK_RATIO" default:"0.1"`
	NumShards       uint64  `env:"NUM_SHARDS" default:"64"`
	SmallBlockLimit uint64  // 计算得出，不从环境变量读取
}

func LoadConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	t := reflect.TypeOf(*cfg)
	v := reflect.ValueOf(cfg).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envName := field.Tag.Get("env")
		defaultVal := field.Tag.Get("default")

		if envName == "" || field.Name == "SmallBlockLimit" {
			continue
		}

		val, ok := os.LookupEnv(envName)
		if !ok {
			val = defaultVal
		}

		switch field.Type.Kind() {
		case reflect.Int:
			intVal, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s must be an integer: %v", envName, err)
			}
			v.Field(i).SetInt(intVal)
		case reflect.Uint64:
			intVal, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s must be an integer: %v", envName, err)
			}
			v.Field(i).SetUint(intVal)
		case reflect.Float64:
			floatVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("%s must be a float: %v", envName, err)
			}
			v.Field(i).SetFloat(floatVal)
		case reflect.String:
			v.Field(i).SetString(val)
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	cfg.calculateDerivedValues()

	return cfg, nil
}

func (c *Config) validate() error {
	if c.TotalSize < c.UnitSize {
		return fmt.Errorf("TOTAL_SIZE must be greater than or equal to UNIT_SIZE")
	}
	// 可以添加更多的验证逻辑
	return nil
}

func (c *Config) calculateDerivedValues() {
	c.SmallBlockLimit = uint64(float64(c.TotalSize) * c.SmallBlockRatio / float64(c.UnitSize))
}
