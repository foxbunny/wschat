package main

import (
	"./command_socket"
	_ "./statik"
	"flag"
	"fmt"
	"github.com/rakyll/statik/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const VERSION = "0.0.5"

var (
	addr    = flag.String("addr", "127.0.0.1:8080", "http service address")
	version = flag.Bool("version", false, "Print the version and exit")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Dreamcatcher chat v%s\n", VERSION)
		os.Exit(0)
	}

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

	fmt.Printf("Dreamcatcher chat v%s\n", VERSION)
	fmt.Println("Starting the server at", *addr)

	http.HandleFunc("/sock", func(w http.ResponseWriter, r *http.Request) {
		command_socket.ServeSock(w, r, cmd)
	})
	http.Handle("/", http.StripPrefix("/", http.FileServer(feAssets)))
	http.ListenAndServe(*addr, nil)
}
