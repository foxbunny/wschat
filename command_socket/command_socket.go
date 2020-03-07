package command_socket

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"strconv"
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

func sockToStdin(ws *websocket.Conn, inputIO chan<- []byte, errIO chan<- Error) {
	ws.SetReadDeadline(time.Now().Add(readWait))
	for {
		log.Println("[SOCKET] Waiting")
		_, msg, err := ws.ReadMessage()
		log.Println("[SOCKET] -> [inputIO]", string(msg))
		if err != nil {
			errIO <- Error{err: err, msg: "Could not read from socket"}
			log.Println("[inputIO] Closing")
			close(inputIO)
			return
		}
		inputIO <- msg
	}
}

func stdoutToSock(ws *websocket.Conn, outputIO <-chan []byte, errIO chan<- Error) {
	for {
		log.Println("[outputIO] Waiting")
		msg, more := <-outputIO
		if more {
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			log.Println("[SOCKET] <- [outputIO]", string(msg))
			if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("[SOCKET] <- [outputIO] GARBLED!")
				if err := ws.WriteMessage(websocket.TextMessage,
					[]byte("Last message was garbled")); err != nil {
					errIO <- Error{err: err, msg: "Could not write to socket"}
				}
				return
			}
		} else {
			log.Println("[outputIO] Done")
			return
		}
	}
}

func ping(ws *websocket.Conn, errIO chan Error, done chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		select {
		case <-ticker.C:
			log.Println("[SOCKET] <- ping")
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				errIO <- Error{err: err, msg: "Could not send ping"}
				return
			}
		case <-done:
			log.Println("[SOCKET] Terminating ping")
			return
		}
	}
}

func logErrors(ws *websocket.Conn, errIO <-chan Error) {
	for {
		err := <-errIO
		log.Println("[ERROR]", err.msg, err.err.Error())
		ws.WriteMessage(websocket.TextMessage, []byte(err.msg))
	}
}

func parseIntParam(q url.Values, param string, def int) int {
	val := q.Get(param)
	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return i
}

func parseFloatParam(q url.Values, param string, def float64) float64 {
	val := q.Get(param)
	n, err := strconv.ParseFloat(val, -1)
	if err != nil {
		return def
	}
	return n
}

func ServeSock(w http.ResponseWriter, r *http.Request, cmd string) {
	log.Println("Starting new connection")

	// Parse out the radio configuration
	q := r.URL.Query()
	params := RadioParams{
		frequency:       parseFloatParam(q, "frequency", DEFAULT_FREQUENCY),
		bandwidth:       parseIntParam(q, "bandwidth", DEFAULT_BANDWIDTH),
		spreadingFactor: parseIntParam(q, "spreadingFactor", DEFAULT_SPREADING_FACTOR),
		codingRate:      parseIntParam(q, "codingRate", DEFAULT_CODING_RATE),
	}

	// Upgrade HTTP connection to websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	// Create channels for communicating with the underlying chat program
	inputIO := make(chan []byte)  // socket -> chat
	outputIO := make(chan []byte) // chat -> socket
	errIO := make(chan Error)
	done := make(chan struct{})

	// Spin up all goroutines
	go SpawnChat(cmd, params, done, inputIO, outputIO, errIO)
	go sockToStdin(ws, inputIO, errIO)
	go stdoutToSock(ws, outputIO, errIO)
	go logErrors(ws, errIO)
	go ping(ws, errIO, done)

	// Block the done channel and wait for something to send to it
	<-done

	// Clean up
	log.Println("[SOCKET] Closing")
	ws.SetWriteDeadline(time.Now().Add(writeWait))
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(closeGracePeriod)
	ws.Close()
	log.Println("[SOCKET] Closed")
}
