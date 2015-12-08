package conf

import (
	"fmt"
	"testing"
)

func TestLoadConf(t *testing.T) {
	conf, err := LoadConf("../r3dashboard.yml")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(conf)
}
