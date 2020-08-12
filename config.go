package main

import (
	"io/ioutil"
	"os"
	"os/user"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen       string                 `yaml:"listen"`
	Secret       string                 `yaml:"secret"`
	User         string                 `yaml:"user"`
	Repositories []string               `yaml:"repositories"`
	SatisConfig  map[string]interface{} `yaml:"satis_config"`

	Container struct {
		Name       string            `yaml:"name"`
		AutoRemove bool              `yaml:"remove"`
		LogType    string            `yaml:"log-type"`
		LogArgs    map[string]string `yaml:"log-args"`
	} `yaml:"container"`

	Directories struct {
		Ssh      string `yaml:"ssh"`
		Composer string `yaml:"composer"`
		Build    string `yaml:"build"`
	} `yaml:"directories"`
}

func GetConfig(location string) (*Config, *user.User, error) {

	out, err := ioutil.ReadFile(location)

	if err != nil {
		return nil, nil, err
	}

	var cnf = new(Config)

	cnf.Listen = ":8080"
	cnf.Container.Name = "composer/satis"
	cnf.Container.AutoRemove = true
	cnf.Container.LogType = "syslog"

	if err := yaml.Unmarshal(out, cnf); err != nil {
		return nil, nil, err
	}

	if cnf.SatisConfig == nil {
		return nil, nil, errors.New("missing satis config.")
	}

	if s := len(cnf.Repositories); s == 0 {
		return nil, nil, errors.New("no repositories configured.")
	} else {
		var respos = make([]map[string]string, s)

		for i := 0; i < s; i++ {
			respos[i] = map[string]string{
				"type": "vcs",
				"name": cnf.Repositories[i],
				"url":  "git@github.com:" + cnf.Repositories[i] + ".git",
			}
		}

		cnf.SatisConfig["repositories"] = respos
	}

	if cnf.Directories.Build == "" {
		dir, err := os.Getwd()

		if err != nil {
			return nil, nil, err
		}

		cnf.Directories.Build = dir + "/build"
	}

	var usr *user.User

	if cnf.User == "" {
		if curr, err := user.Current(); err != nil {
			return nil, nil, err
		} else {
			usr = curr
		}
	} else {
		if curr, err := user.Lookup(cnf.User); err != nil {
			return nil, nil, err
		} else {
			usr = curr
		}
	}

	if cnf.Directories.Ssh == "" {
		cnf.Directories.Ssh = usr.HomeDir + "/.ssh"
	}

	return cnf, usr, nil
}
