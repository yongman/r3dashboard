package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ksarch-saas/r3dashboard/conf"
	"github.com/ksarch-saas/r3dashboard/inspector"
	"golang.org/x/net/websocket"
)

func appStatus(ws *websocket.Conn) {
	fmt.Println("ws:new client")
	firstrun := true
	c := inspector.ClientRegist(ws)
	for {
		if firstrun {
			firstrun = false
			inspector.FeedClientWithObsoleteInfo(c)
		} else {
			appInfoMap := <-c.C
			//choose the info we care
			data, err := json.Marshal(appInfoMap)
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, err = ws.Write([]byte(data))
			if err != nil {
				break
			}
		}
	}
	inspector.ClientRemove(c)
	fmt.Println("ws:client closed")
}

func RunServer(meta *conf.DashboardConf) {
	//handle websocket
	http.Handle("/status", websocket.Handler(appStatus))
	//handle http request
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/ui/", http.StripPrefix("/ui/", fs))
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "PONG")
	})

	err := http.ListenAndServe(meta.Listen, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
