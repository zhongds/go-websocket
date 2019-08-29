package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:4042", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func handleConnect(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()
	if len(vals["name"]) == 0 {
		log.Fatal("参数name是必须的")
		return
	}

	name := vals["name"][0]
	if clients[name] != nil {
		data := &ReturnBody{Body: name + "已经被占用，请换一个"}
		b, _ := json.Marshal(data)
		w.Write(b)
		w.WriteHeader(500)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade err:", err)
		return
	}

	client := generateClient(c, name)
	client.register()

	log.Print("=====已经注册的客户端 start=======")
	for k, _ := range clients {
		log.Printf("客户端：%v", k)
	}
	log.Print("=====已经注册的客户端 end=======")

	defer func() {
		log.Println("===defer=====")
		client.unregister()
	}()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			break
		}
		var messageBody MessageBody
		err = json.Unmarshal(message, &messageBody)
		if err != nil {
			c.WriteJSON(&ReturnBody{Body: "参数格式不对: " + err.Error()})
			break
		}
		log.Printf("recv: %s", message)
		err = client.handleMessage(messageBody)
		if err != nil {
			log.Println("write err:", err)
			break
		}
	}
}

func handleTest(w http.ResponseWriter, r *http.Request) {
	log.Println("====http request: /test======")
}

func main() {
	log.Println("=====start=====")
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/ws", handleConnect)
	http.HandleFunc("/test", handleTest)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
