package inspector

import (
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
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
	//分片最大大小
	ReplicaMax string
	//分片最小大小
	ReplicaMin string
	//分片平均大小
	ReplicaAvg string
}

var wsClients []*Client

var appInfoMap map[string]*AppInfo

func Run(meta *conf.DashboardConf) {
	tickC := time.NewTicker(time.Second * 10).C

	count := 0
	var children []string
	zkaddr := meta.Zk
	meta_server := meta.Meta_server
	inner := func() {
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
				//frequency 1min
				//update apps
				inner()
			}
			count = (count + 1) % 6

			//if have no web client just sleep
			if len(wsClients) == 0 {
				continue
			}

			conn, err := redis.DialTimeout("tcp", meta_server, 10*time.Second, 10*time.Second, 10*time.Second)
			if err != nil {
				fmt.Println(err)
				continue
			}
			//frequency 10s
			aim := make(map[string]*AppInfo, 500)
			for _, app := range children {
				rsss, err := FetchReplicaSets(app, zkaddr)
				if err != nil {
					fmt.Println(err)
				}
				cc := context.GetControllerConfig()
				m := map[string]string{
					"ip":        cc.Ip,
					"http_port": fmt.Sprintf("%d", cc.HttpPort),
					"ws_port":   fmt.Sprintf("%d", cc.WsPort),
				}
				key := fmt.Sprintf("meta_%s", app)
				_, err = conn.Do("HMSET", redis.Args{}.Add(key).AddFlat(m)...)
				if err != nil {
					fmt.Println(err)
					return
				}

				appinfo := AppCheck(rsss)
				aim[app] = appinfo
			}
			appInfoMap = aim
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
	var replicaMax int64
	var replicaMin int64
	var replicaTotal int64

	first := true
	totalNodes = 0
	replicaEqual = true
	replicaMin = 1024 * 1024 * 1024 * 1024 //1TB
	replicaMax = -1
	replicaTotal = 0

	for _, rs := range rss {
		//get used memory from master
		if rs.Master != nil && rs.Master.IsArbiter() {
			continue
		}
		if rs.Master != nil {
			usedMemory := rs.Master.UsedMemory
			if replicaMax < usedMemory {
				replicaMax = usedMemory
			}
			if replicaMin > usedMemory {
				replicaMin = usedMemory
			}
			replicaTotal += usedMemory
		}
		totalNodes += len(rs.AllNodes())
		if first {
			first = false
			partitions = len(rss)
			replicas = len(rs.AllNodes())

		} else {
			//check replicaset if has same number nodes
			if replicas != len(rs.AllNodes()) {
				replicaEqual = false
			}
		}
	}
	replicaAvg := 0.0
	if len(rss) > 0 {
		replicaAvg = float64(replicaTotal / int64(len(rss)))
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
		ReplicaMax:   fmt.Sprintf("%0.2f", float64(replicaMax)/1024.0/1024.0/1024.0),
		ReplicaMin:   fmt.Sprintf("%0.2f", float64(replicaMin)/1024.0/1024.0/1024.0),
		ReplicaAvg:   fmt.Sprintf("%0.2f", replicaAvg/1024.0/1024.0/1024.0),
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
