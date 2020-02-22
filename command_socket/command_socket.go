package command_socket

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	// Maximum allowed wait time for writes to the client
	writeWait = 10 * time.Second

	// Maximum allowed wait time for reads from the client
	readWait = 60 * time.Second

	// Time to wait before forcibly disconnecting clients
	closeGracePeriod = 10 * time.Second

	// Period after which the pong read is timed-out
	pongWait = readWait

	// Interval in which to perform the pings (should be less than pongWait)
	pingInterval = 50 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func sockToStdin(ws *websocket.Conn, cmdIO chan<- []byte, errIO chan<- Error) {
	ws.SetReadDeadline(time.Now().Add(readWait))
	defer ws.Close()
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			errIO <- Error{err: err, msg: "Could not read from socket"}
			break
		}
		message = append(message, '\n')
		cmdIO <- message
	}
}

func stdoutToSock(ws *websocket.Conn, cmdIO <-chan []byte,
	errIO chan<- Error, done chan struct{}) {
	for {
		ws.SetWriteDeadline(time.Now().Add(writeWait))
		msg := <-cmdIO
		if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			ws.Close()
			errIO <- Error{err: err, msg: "Could not write to socket"}
			break
		}
	}
	close(done)
	ws.SetWriteDeadline(time.Now().Add(writeWait))
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(closeGracePeriod)
	ws.Close()
}

func ping(ws *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		select {
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				log.Println("ping:", err)
			}
		case <-done:
			break
		}
	}
}

func logErrors(ws *websocket.Conn, errIO <-chan Error, done chan struct{}) {
	for {
		err := <-errIO
		log.Println(err.msg, err.err.Error())
		ws.WriteMessage(websocket.TextMessage, []byte(err.msg))
	}
	close(done)
}

func ServeSock(w http.ResponseWriter, r *http.Request, cmd string) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer ws.Close()

	cmdIO := make(chan []byte)
	errIO := make(chan Error)
	done := make(chan struct{})
	params := RadioParams{
		frequency:       1000,
		spreadingFactor: 192,
		bandwidth:       200,
		codingRate:      5,
	}

	go SpawnChat(cmd, params, done, cmdIO, errIO)
	go sockToStdin(ws, cmdIO, errIO)
	go stdoutToSock(ws, cmdIO, errIO, done)
	go logErrors(ws, errIO, done)
	go ping(ws, done)

	<-done
}
