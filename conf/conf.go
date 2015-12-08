package conf

import (
	"gopkg.in/yaml.v1"
	"io/ioutil"
)

type DashboardConf struct {
	Listen string `yaml:"listen,omitempty"`
	Zk     string `yaml:"zk,omitempty"`
}

func LoadConf(filename string) (*DashboardConf, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	dc := &DashboardConf{}
	err = yaml.Unmarshal(content, dc)
	if err != nil {
		return nil, err
	}
	return dc, nil
}
