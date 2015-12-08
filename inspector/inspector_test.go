package inspector

import (
	c "../conf"
	"fmt"
	"testing"
)

func TestRun(t *testing.T) {
	meta, err := c.LoadConf("../redis-monitor.yml")
	if err != nil {
		return
	}
	fmt.Println(meta)
	Run(meta)
}
