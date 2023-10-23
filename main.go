package main

import (
	"crypto/tls"
	"encoding/json"
	"github.com/nxtrace/NTrace-core/util"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/chzyer/readline"
	"github.com/gorilla/websocket"
	"github.com/nxtrace/wscat-go/pow"
)

func probe() bool {
	fastIp, host, port := "127.0.0.1", "api.leo.moe", "443"
	jwtToken, ua := util.EnvToken, []string{"Privileged Client"}
	err := error(nil)
	if jwtToken == "" {
		if util.GetPowProvider() == "" {
			jwtToken, err = pow.GetToken(fastIp, host, port)
		} else {
			jwtToken, err = pow.GetToken(util.GetPowProvider(), util.GetPowProvider(), port)
		}
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		ua = []string{"wscat-go"}
	}
	log.Println("PoW Start")

	if err != nil {
		log.Println("连接失败:", err)
		return false
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
		log.Println("连接失败:", err)
		return false
	}
	defer func(c *websocket.Conn) {
		err := c.Close()
		if err != nil {
			log.Println(err)
		}
	}(c)

	log.Println("LeoMoeAPI V2 连接成功！")
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer func(rl *readline.Instance) {
		err := rl.Close()
		if err != nil {
			log.Println(err)
		}
	}(rl)

	probeIP := func(ENV_FIELD, defaultIP string) bool {
		for i := 0; i < 3; i++ {
			ip := util.GetenvDefault(ENV_FIELD, defaultIP)
			err = c.WriteMessage(websocket.TextMessage, []byte(ip))
			if err != nil {
				log.Println("发送失败:", err)
				continue
			}

			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("接收失败:", err)
				continue
			}

			var ipObj map[string]interface{}
			err = json.Unmarshal(message, &ipObj)
			if err != nil {
				log.Println("JSON解析失败:", err)
				continue
			}

			// New colorjson Formatter
			f := colorjson.NewFormatter()
			f.Indent = 2

			s, _ := f.Marshal(ipObj)
			log.Println(string(s))
			return true
		}
		return false
	}

	flagV4 := probeIP("V4IP", "1.1.1.1")
	flagV6 := probeIP("V6IP", "2400:3200::1")

	if flagV4 && flagV6 {
		return true
	} else {
		return false
	}
}

func main() {
	result := probe()
	if result {
		log.Println("Probe succeeded.")
	} else {
		log.Println("Probe failed.")
	}
}
