package interfaces

import (
	"net"
	"sync"
)

type Server struct {
	Address     string
	Connections map[string]*User
	IpAddresses map[string]*User
	Rooms       map[string]*Room
	Messages    chan Message
	Mutex       sync.Mutex
}

type Message struct {
	SenderId       string
	SenderUsername string
	Content        string
	Timestamp      string
	RoomId         string
}

type User struct {
	UserId        string
	Username      string
	StoreFilePath string
	Conn          net.Conn
	IsOnline      bool
	IpAddress     string
	CurrentRoom   string
}

type Room struct {
	RoomId      string
	RoomName    string
	Creator     string
	Participants map[string]*User
	Messages    chan Message
	CreatedAt   string
	Mutex       sync.Mutex
}
