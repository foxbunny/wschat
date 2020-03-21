package command_socket

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"unicode/utf8"
)

var GARBLED = errors.New("garbled")

func stdoutToOutput(r io.ReadCloser, outputIO chan<- []byte,
	errorIO chan<- Error) {
	defer r.Close()
	log.Println("[STDOUT] Waiting")
	s := bufio.NewScanner(r)
	for s.Scan() {
		msg := s.Bytes()
		if utf8.Valid(msg) {
			log.Println("[outputIO] <- [STDOUT]", string(msg))
			outputIO <- msg
		} else {
			errorIO <- Error{err: GARBLED, msg: "Last message was garbled"}
			log.Println("[STDOUT] Invalid message received")
		}
	}
	log.Println("No more messages to send")
	if s.Err() != nil {
		errorIO <- Error{err: s.Err(), msg: "Cannot read from chat program"}
		log.Println("[outputIO] Closing")
		close(outputIO)
	}
	log.Println("[STDOUT] Done")
}

func inputToStdin(w io.WriteCloser, inputIO <-chan []byte, errorIO chan<- Error) {
	defer w.Close()
	for {
		log.Println("[inputIO] Waiting")
		msg, more := <-inputIO
		if more {
			log.Println("[inputIO] -> [STDIN]", string(msg))
			msg = append(msg, '\n')
			if _, err := w.Write(msg); err != nil {
				errorIO <- Error{err: err, msg: "Cannot send to chat program"}
				return
			}
			log.Println("[STDIN] Wrote")
		} else {
			log.Println("[STDIN] Done")
			return
		}
	}
}

func SpawnChat(
	cmd string,
	params RadioParams,
	done chan struct{},
	inputIO chan []byte,
	outputIO chan []byte,
	errIO chan Error) {

	defer close(done)

	// Create a common output pipe
	outr, outw, err := os.Pipe()
	if err != nil {
		errIO <- Error{err: err, msg: "Failed to open common output pipe"}
		close(done)

	}

	// Start the command and bind to input/output pipes
	proc := exec.Command(
		cmd,
		"-f",
		strconv.FormatFloat(params.frequency, 'f', -1, 32),
		"-b",
		strconv.Itoa(Bandwidths[params.bandwidth]),
		"-s",
		strconv.Itoa(SpreadingFactors[params.spreadingFactor]),
		"-c",
		strconv.Itoa(CodingRates[params.codingRate]),
	)
	inw, err := proc.StdinPipe()
	if err != nil {
		errIO <- Error{err: err, msg: "Failed to open input pipe for command"}
		close(done)
		return
	}
	proc.Stdout = outw
	proc.Stderr = outw
	if err = proc.Start(); err != nil {
		errIO <- Error{err: err, msg: "Could not start the process"}
		close(done)
	}

	log.Println("[CMD] Spawned process", proc.Process.Pid, cmd, proc.Args)

	go stdoutToOutput(outr, outputIO, errIO)
	inputToStdin(inw, inputIO, errIO)

	if err := proc.Process.Signal(os.Interrupt); err != nil {
		log.Println("[PROC] Cannot interrupt process")
		if err := proc.Process.Signal(os.Kill); err != nil {
			log.Println("[PROC] Cannot kill process")
		}
	}

	if _, err := proc.Process.Wait(); err != nil {
		log.Println("[PROC] Chat program terminated with error")
	}

	log.Println("[PROC] Done")
}
