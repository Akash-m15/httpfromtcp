package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:42069")
	if err != nil {
		log.Fatalf("Error while resolving udp address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Error while listening: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(">")
		str, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading String: %v", err)
		}

		str = strings.TrimSpace(str)
		_, err = conn.Write([]byte(str + "\n"))
		if err != nil {
			log.Fatalf("error writing to port through udp: %v", err)
		}
	}
}
