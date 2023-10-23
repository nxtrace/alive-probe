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

func main() {
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
		log.Println("#### 死了 ####")
		return
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
		log.Println("#### 死了 ####")
		return
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

	flagV4 := false
	v4IP := util.GetenvDefault("V4IP", "1.1.1.1")
	for i := 0; i < 3; i++ {
		err = c.WriteMessage(websocket.TextMessage, []byte(v4IP))
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
		flagV4 = true
		break
	}
	flagV6 := false
	v6IP := util.GetenvDefault("V6IP", "2400:3200::1")
	for i := 0; i < 3; i++ {
		err = c.WriteMessage(websocket.TextMessage, []byte(v6IP))
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
		flagV6 = true
		break
	}
	if flagV4 && flagV6 {
		log.Println("#### 存活 ####")
	} else {
		log.Println("#### 死了 ####")
	}
}
