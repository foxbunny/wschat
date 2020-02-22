package command_socket

import (
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

type RadioParams struct {
	frequency       float64
	spreadingFactor int
	bandwidth       int
	codingRate      int
}

func stdOutToChn(r io.Reader, cmdIO chan []byte, errorIO chan Error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		msg := s.Bytes()
		cmdIO <- msg
	}
	if s.Err() != nil {
		errorIO <- Error{err: s.Err(), msg: "Cannot read from STDOUT"}
	}
}

func chanToStdIn(w io.Writer, cmdIO chan []byte, errorIO chan Error) {
	for {
		message := <-cmdIO
		message = append(message, '\n')
		if _, err := w.Write(message); err != nil {
			errorIO <- Error{err: err, msg: "Cannot write to STDIN"}
		}
	}
}

func SpawnChat(
	cmd string,
	params RadioParams,
	done chan struct{},
	cmdIO chan []byte,
	errIO chan Error) {

	defer close(done)

	// Output read/write pipes
	outr, outw, err := os.Pipe()
	if err != nil {
		return
	}
	defer outr.Close()
	defer outw.Close()

	// Input read/write pipes
	inr, inw, err := os.Pipe()
	if err != nil {
		return
	}
	defer inr.Close()
	defer inw.Close()

	args := []string{
		"-f",
		strconv.FormatFloat(params.frequency, 'f', -1, 32),
		"-s",
		strconv.Itoa(params.spreadingFactor),
		"-b",
		strconv.Itoa(params.bandwidth),
		"-c",
		strconv.Itoa(params.codingRate),
	}

	// Start the command and bind to input/output pipes
	proc, err := os.StartProcess(cmd, args, &os.ProcAttr{
		Files: []*os.File{inr, outw, outw},
	})
	if err != nil {
		errIO <- Error{err: err, msg: "Failed to spawn command"}
	}

	go chanToStdIn(inw, cmdIO, errIO)
	go stdOutToChn(outr, cmdIO, errIO)

	if err := proc.Signal(os.Interrupt); err != nil {
		log.Println("term:", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		if err := proc.Signal(os.Kill); err != nil {
			log.Println("kill:", err)
		}
		<-done
	}

	if _, err := proc.Wait(); err != nil {
		log.Println("wait:", err)
	}
}
