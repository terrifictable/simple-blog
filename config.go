package main

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AllowLogin bool `yaml:"allow_login,omitempty"`
	DevMode    bool `yaml:"dev_mode,omitempty"`

	Prefix string `yaml:"prefix,omitempty"`

	DB struct {
		User     string `yaml:"user,omitempty"`
		Password string `yaml:"password,omitempty"`
		Address  string `yaml:"address,omitempty"`
		DBName   string `yaml:"db_name,omitempty"`
	} `yaml:"db,omitempty"`
}

var defaults = struct {
	AllowLogin bool
	Prefix     string
	DB         struct {
		User     string
		Password string
		Address  string
		DBName   string
	}
}{
	AllowLogin: true,
	Prefix:     "./",
	DB: struct {
		User     string
		Password string
		Address  string
		DBName   string
	}{
		User:     string(rune(0x0)),
		Password: string(rune(0x0)),
		Address:  string(rune(0x0)),
		DBName:   string(rune(0x0)),
	},
}

func (cfg *Config) ParseConfig() error {
	parse := func(out *string, name string, def string) error {
		var exist bool
		*out, exist = os.LookupEnv(name)
		if !exist {
			fmt.Printf("Couldnt find '%s' in environment, using default '%s'\n", name, def)
			if def == string(rune(0x0)) {
				return fmt.Errorf("no default value for '%s'", name)
			}
			*out = def
		}
		return nil
	}

	var err error

	var allow_login string
	MustEmpty(parse(&allow_login, "ALLOW_LOGIN", strconv.FormatBool(defaults.AllowLogin)))
	cfg.AllowLogin, err = strconv.ParseBool(allow_login)
	if err != nil {
		return err
	}

	MustEmpty(parse(&cfg.Prefix, "PREFIX", defaults.Prefix))
	var dev string
	MustEmpty(parse(&dev, "DEVMODE", "false"))
	cfg.DevMode, err = strconv.ParseBool(dev)
	if err != nil {
		return err
	}

	MustEmpty(parse(&cfg.DB.User, "DB_USER", defaults.DB.User))
	MustEmpty(parse(&cfg.DB.Password, "DB_PASSWORD", defaults.DB.Password))
	MustEmpty(parse(&cfg.DB.Address, "DB_ADDR", defaults.DB.Address))
	MustEmpty(parse(&cfg.DB.DBName, "DB_NAME", defaults.DB.DBName))

	return nil
}
