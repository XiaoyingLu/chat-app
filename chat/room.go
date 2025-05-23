package main

import (
	"log"
	"net/http"

	"trace"

	"github.com/gorilla/websocket"
	"github.com/stretchr/objx"
)

// room type will be responsible for managing client connections and routing messages in and out
type room struct {
	// forward is a channel that holds incoming messages that should be forwarded to the other clients
	forward chan *message
	// join is a channel for clients wishing to join the room
	join chan *client
	// leave is a channel for clients wishing to leave the room
	leave chan *client
	// clients holds all current clients in this room
	clients map[*client]bool // low-memory way of storing the reference
	// tracer will receive trace information of activity in this room
	tracer trace.Tracer
	// avatar is the avatar information will be obtained
	avatar Avatar
}


func newRoom(avatar Avatar) *room {
	return &room{
		forward: make(chan *message),
		join: make(chan *client),
		leave: make(chan *client),
		clients: make(map[*client]bool),
		tracer: trace.Off(),
		avatar: avatar,
	}
}

func (r *room) run() {
	for {
		// use select statements whenever we need to synchronize or modify shared memory
		select {
		case client := <-r.join:
			// joining
			r.clients[client] = true
			r.tracer.Trace("New client joined")
		case client := <-r.leave:
			// leaving
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("Client left")
		case msg := <-r.forward:
			// forward message to all clients
			for client := range r.clients {
				client.send <- msg
				r.tracer.Trace("Message received: ", msg.Message)
			}
		}
	}
}


const (
	socketBufferSize = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}
func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil) // upgrades the HTTP server connection to the WebSocket protocol
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	authCookie, err := req.Cookie("auth")
	if err != nil {
		log.Fatal("Failed to retrieve auth cookie:", err)
		return
	}
	client := &client{
		socket: socket,
		send: make(chan *message, messageBufferSize),
		room: r,
		userData: objx.MustFromBase64(authCookie.Value),
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}
