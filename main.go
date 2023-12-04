package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nxtrace/NTrace-core/util"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/gorilla/websocket"
	"github.com/nxtrace/wscat-go/pow"
)

func probe() (bool, string) {
	var logs strings.Builder

	// 使用这个函数代替直接的 log.Println
	logFunc := func(msg ...interface{}) {
		logs.WriteString(fmt.Sprintln(msg...))
	}
	// 由于本仓库用途为探针，所以仅考虑本机IP(不套CDN的情况)，SNI/HOST直接使用api.nxtrace.org
	fastIp, host, port := "127.0.0.1", "api.nxtrace.org", "443"
	jwtToken, ua := util.EnvToken, []string{"Privileged Client"}
	err := error(nil)
	if jwtToken == "" {
		if util.GetPowProvider() == "" {
			jwtToken, err = pow.GetToken(fastIp, host, port)
		} else {
			jwtToken, err = pow.GetToken(util.GetPowProvider(), util.GetPowProvider(), port)
		}
		if err != nil {
			logFunc(err)
			return false, logs.String()
		}
		ua = []string{"wscat-go"}
	}
	logFunc("PoW Start")

	if err != nil {
		logFunc("连接失败:", err)
		return false, logs.String()
	}

	requestHeader := http.Header{
		"Authorization": []string{"Bearer " + jwtToken},
		"Host":          []string{host},
		"User-Agent":    ua,
	}
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		ServerName: host,
	}
	proxyUrl := util.GetProxy()
	if proxyUrl != nil {
		dialer.Proxy = http.ProxyURL(proxyUrl)
	}
	u := url.URL{Scheme: "wss", Host: fastIp + ":" + port, Path: "/v3/ipGeoWs"}

	websocket.DefaultDialer.HandshakeTimeout = time.Second * 3
	c, _, err := websocket.DefaultDialer.Dial(u.String(), requestHeader)
	if err != nil {
		logFunc("连接失败:", err)
		return false, logs.String()
	}
	defer func(c *websocket.Conn) {
		err := c.Close()
		if err != nil {
			logFunc(err)
		}
	}(c)

	logFunc("LeoMoeAPI V2 连接成功！")
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer func(rl *readline.Instance) {
		err := rl.Close()
		if err != nil {
			logFunc(err)
		}
	}(rl)

	probeIP := func(ENV_FIELD, defaultIP string) bool {
		for i := 0; i < 3; i++ {
			ip := util.GetenvDefault(ENV_FIELD, defaultIP)
			err = c.WriteMessage(websocket.TextMessage, []byte(ip))
			if err != nil {
				logFunc("发送失败:", err)
				continue
			}

			_, message, err := c.ReadMessage()
			if err != nil {
				logFunc("接收失败:", err)
				continue
			}

			var ipObj map[string]interface{}
			err = json.Unmarshal(message, &ipObj)
			if err != nil {
				logFunc("JSON解析失败:", err)
				continue
			}

			// New colorjson Formatter
			//f := colorjson.NewFormatter()
			//f.Indent = 2
			//
			//s, _ := f.Marshal(ipObj)
			//logFunc(string(s))
			//return true

			// 替换原先的colorjson代码
			formattedJSON, err := json.MarshalIndent(ipObj, "", "  ") // 使用两个空格进行缩进
			if err != nil {
				logFunc("JSON格式化失败:", err)
				continue
			}
			logFunc(string(formattedJSON))
			return true
		}
		return false
	}

	flagV4 := probeIP("V4IP", "1.1.1.1")
	flagV6 := probeIP("V6IP", "2400:3200::1")

	if flagV4 && flagV6 {
		return true, logs.String()
	} else {
		return false, logs.String()
	}
}

func main() {
	hostPort := util.GetenvDefault("PROBE_HOSTPORT", "127.0.0.1:8080")

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		alive, logMessage := probe()

		// 同步在服务端打印日志消息
		log.Println(logMessage)

		if alive {
			c.JSON(http.StatusOK, gin.H{
				"status":  "alive",
				"message": "Probe succeeded.",
				"log":     logMessage,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "dead",
				"message": "Probe failed.",
				"log":     logMessage,
			})
		}
	})

	err := r.Run(hostPort)
	if err != nil {
		log.Printf("Failed to run server: %v", err)
	}
}
