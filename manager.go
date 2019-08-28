package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

type MessageBody struct {
	Content string
	To      string // name => 私聊
	Room    string // 房间
}

type Client struct {
	Id     string
	Name   string // 用户名, 可以换成uid, 用来识别client
	Socket *websocket.Conn
	status int // 预留字段 状态：在线/离线
}

type Room struct {
	Name    string
	Clients []*Client
}

type ReturnBody struct {
	Sender   string
	Receiver string
	Body     string
}

// TODO 加入房间/离开房间 连接成功/退出连接 私聊

var clients = make(map[string]*Client)
var rooms = make(map[string]*Room)

// 生成client
func generateClient(c *websocket.Conn, name string) *Client {
	uid := uuid.Must(uuid.NewV4()).String()
	client := &Client{Id: uid, Socket: c, Name: name}
	return client
}

// 连接成功
func (client *Client) register() {
	clients[client.Name] = client
}

// 退出连接
func (client *Client) unregister() error {
	log.Printf("客户端%v退出连接", client.Name)
	delete(clients, client.Name)
	err := client.Socket.Close()
	if err != nil {
		log.Println("@@@close err", err.Error())
	}
	return err
}

func (client *Client) handleMessage(msg MessageBody) error {
	if msg.To != "" {
		return client.send(&msg)
	}
	if msg.Room != "" {
		return client.broadcastToRoom(msg.Room, &msg)
	}
	return errors.New("参数必须有To或者Room字段")
}

// join room
func (client *Client) joinRoom(roomName string) *Room {
	room := rooms[roomName]
	log.Println("加入房间: ", roomName)
	if room == nil {
		newRoom := &Room{Name: roomName, Clients: []*Client{client}}
		rooms[roomName] = newRoom
		return newRoom
	}
	var isExist bool
	for _, item := range room.Clients {
		if item.Id == client.Id {
			isExist = true
			break
		}
	}
	if !isExist {
		room.Clients = append(room.Clients, client)
	}
	client.broadcastToRoom(roomName, &ReturnBody{Body: fmt.Sprintf("%v进入了房间", client.Name)})
	return room
}

// romove room
func (client *Client) removeRoom(roomName string) error {
	room := rooms[roomName]
	if room == nil {
		return errors.New(fmt.Sprintf("%v房间不存在", roomName))
	}

	index := -1
	for i, item := range room.Clients {
		if item.Id == client.Id {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New(fmt.Sprintf("%v不在房间", client.Name))
	}
	room.Clients = append(room.Clients[:index], room.Clients[index+1:]...)
	data := &ReturnBody{
		Body:   fmt.Sprintf("%v离开了房间", client.Name),
		Sender: client.Name,
	}
	client.broadcastToRoom(roomName, data)
	return nil
}

func (client *Client) send(messageBody *MessageBody) error {
	target := clients[messageBody.To] // client
	if target == nil {
		return errors.New("客户端不存在")
	}
	data := &ReturnBody{
		Body:     messageBody.Content,
		Receiver: messageBody.To,
		Sender:   client.Name,
	}
	return target.Socket.WriteJSON(data)
}

// 广播  但是不发送给自己
func (client *Client) broadcastToRoom(roomName string, msg interface{}) error {
	room := rooms[roomName]
	if room == nil {
		return errors.New(fmt.Sprintf("%v房间不存在", roomName))
	}
	clients := room.Clients
	var errmsg string
	for _, item := range clients {
		if item.Id != client.Id {
			err := item.Socket.WriteJSON(msg)
			if err != nil {
				errmsg += fmt.Sprintf("%v客户端发送消息失败: %v", item.Id, err.Error())
			}
		}
	}
	if errmsg != "" {
		return errors.New(errmsg)
	}
	return nil
}
