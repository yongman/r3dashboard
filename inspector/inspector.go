package inspector

import (
	"fmt"
	"time"

	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/controller/command"
	"github.com/ksarch-saas/cc/frontend/api"
	m "github.com/ksarch-saas/cc/meta"
	"github.com/ksarch-saas/cc/utils"
	"github.com/ksarch-saas/r3dashboard/conf"
	"golang.org/x/net/websocket"
)

type Client struct {
	C  chan map[string]*AppInfo
	Ws *websocket.Conn
}

type AppInfo struct {
	TotalNodes int
	//分片数
	Partitions int
	//副本数
	Replicas int
	//异常实例数
	Exceptions int
	//分片是否对称
	ReplicaEqual bool
}

var wsClients []*Client

var appInfoMap map[string]*AppInfo

func Init() {
	wsClients = make([]*Client, 2)
}

func Run(meta *conf.DashboardConf) {
	tickC := time.NewTicker(time.Second * 30).C

	count := 0
	var children []string
	zkaddr := meta.Zk
	inner := func() {
		appInfoMap = make(map[string]*AppInfo, 100)
		zconn, _, err := m.DialZk(zkaddr)
		if err != nil {
			fmt.Println("zk: dial zookeeper failed")
		}
		children, _, err = zconn.Children("/r3/app")
		if err != nil {
			fmt.Println("zk: call children failed ", err)
		}
		defer func() {
			if zconn != nil {
				zconn.Close()
			}
		}()
	}

	inner()

	for {
		select {
		case <-tickC:
			if count == 0 {
				//frequency 3min
				//update apps
				inner()
			}
			count = (count + 1) % 6

			//if have no web client just sleep
			if len(wsClients) == 0 {
				continue
			}

			//frequency 30s
			for _, app := range children {
				rsss, err := FetchReplicaSets(app, zkaddr)
				if err != nil {
					fmt.Println(err)
				}
				appinfo := AppCheck(rsss)
				appInfoMap[app] = appinfo
			}
			//send the appinfomap to websocket
			for _, c := range wsClients {
				c.C <- appInfoMap
			}
		}
	}
}

func FeedClientWithObsoleteInfo(c *Client) {
	if appInfoMap != nil && len(appInfoMap) != 0 {
		c.C <- appInfoMap
	}
}

func FetchReplicaSets(app string, zkaddr string) (command.FetchReplicaSetsResult, error) {
	//get controller address
	err := context.SetApp(app, zkaddr)
	ctrl_addr := context.GetLeaderAddr()
	url := "http://" + ctrl_addr + api.FetchReplicaSetsPath
	//send http request to controler get nodes info
	resp, err := utils.HttpGet(url, nil, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return command.FetchReplicaSetsResult{}, err
	}
	var rss command.FetchReplicaSetsResult
	err = utils.InterfaceToStruct(resp.Body, &rss)
	if err != nil {
		fmt.Println(err)
		return command.FetchReplicaSetsResult{}, err
	}
	return rss, nil
}

func AppCheck(rsss command.FetchReplicaSetsResult) *AppInfo {
	rss := rsss.ReplicaSets
	nss := rsss.NodeStates

	var totalNodes int    //总节点
	var partitions int    //分片数
	var replicas int      //副本数
	var notRunning int    //异常实例数
	var replicaEqual bool //各副本节点数是否相同

	first := true
	totalNodes = 0
	replicaEqual = true

	for _, rs := range rss {
		totalNodes += len(rs.AllNodes())
		if first {
			first = false
			partitions = len(rss)
			replicas = len(rs.AllNodes())

		} else {
			//check replicaset if has same number nodes
			if replicas != len(rs.AllNodes()) {
				replicaEqual = false
				break
			}
		}
	}
	notRunning = 0
	for _, ns := range nss {
		if ns != "RUNNING" {
			notRunning++
		}
	}
	appinfo := AppInfo{
		TotalNodes:   totalNodes,
		Partitions:   partitions,
		Replicas:     replicas,
		Exceptions:   notRunning,
		ReplicaEqual: replicaEqual,
	}
	return &appinfo
}

func ClientRegist(ws *websocket.Conn) *Client {
	client := &Client{
		C:  make(chan map[string]*AppInfo, 100),
		Ws: ws,
	}
	wsClients = append(wsClients, client)

	return client
}

func ClientRemove(cli *Client) {
	for i, c := range wsClients {
		if c == cli {
			wsClients = append(wsClients[:i], wsClients[i+1:]...)
			break
		}
	}
}
