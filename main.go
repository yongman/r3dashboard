package main

import (
	"fmt"

	"github.com/ksarch-saas/r3dashboard/conf"
	"github.com/ksarch-saas/r3dashboard/inspector"
	"github.com/ksarch-saas/r3dashboard/server"
)

func main() {
	//read meta config
	meta, err := conf.LoadConf("./r3dashboard.yml")
	if err != nil {
		fmt.Println(err)
		return
	}

	go inspector.Run(meta)
	server.RunServer(meta)
}
