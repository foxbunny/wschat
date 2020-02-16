package main

import (
	_ "./statik"
	"bufio"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/rakyll/statik/fs"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

var (
	addr = flag.String("addr", "127.0.0.1:8080", "http service address")
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

func sockToStdin(ws *websocket.Conn, w io.Writer) {
	ws.SetReadDeadline(time.Now().Add(readWait))
	defer ws.Close()
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}
		fmt.Println(string(message))
		message = append(message, '\n')
		if _, err := w.Write(message); err != nil {
			break
		}
	}
}

func stdoutToSock(ws *websocket.Conn, r io.Reader, done chan struct{}) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		ws.SetWriteDeadline(time.Now().Add(writeWait))
		msg := s.Bytes()
		fmt.Println(string(msg))
		if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			ws.Close()
			break
		}
	}
	if s.Err() != nil {
		log.Println("scan:", s.Err())
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

func internalError(ws *websocket.Conn, msg string, err error) {
	log.Println(msg, err)
	ws.WriteMessage(websocket.TextMessage, []byte("Internal server error."))
}

func serveSock(w http.ResponseWriter, r *http.Request, cmd string) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer ws.Close()

	// Output read/write pipes
	outr, outw, err := os.Pipe()
	if err != nil {
		internalError(ws, "out:", err)
		return
	}
	defer outr.Close()
	defer outw.Close()

	// Input read/write pipes
	inr, inw, err := os.Pipe()
	if err != nil {
		internalError(ws, "in:", err)
		return
	}
	defer inr.Close()
	defer inw.Close()

	proc, err := os.StartProcess(cmd, []string{}, &os.ProcAttr{
		Files: []*os.File{inr, outw, outw},
	})
	if err != nil {
		internalError(ws, "cmd:", err)
		return
	}

	stdoutDone := make(chan struct{})

	go stdoutToSock(ws, outr, stdoutDone)
	go ping(ws, stdoutDone)

	sockToStdin(ws, inw)

	if err := proc.Signal(os.Interrupt); err != nil {
		log.Println("term:", err)
	}

	select {
	case <-stdoutDone:
	case <-time.After(time.Second):
		if err := proc.Signal(os.Kill); err != nil {
			log.Println("kill:", err)
		}
		<-stdoutDone
	}

	if _, err := proc.Wait(); err != nil {
		log.Println("wait:", err)
	}
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatal("You must specify the command to run")
	}
	cmd, err := exec.LookPath(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	feAssets, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting the server at", *addr)

	http.HandleFunc("/sock", func(w http.ResponseWriter, r *http.Request) {
		serveSock(w, r, cmd)
	})
	http.Handle("/", http.StripPrefix("/", http.FileServer(feAssets)))
	http.ListenAndServe(*addr, nil)
}
