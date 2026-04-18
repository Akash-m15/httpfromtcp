package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/Akash-m15/httpfromtcp/internal/request"
)

func getLinesFromChannel(f io.ReadCloser) <-chan string {
	lineChan := make(chan string, 1)

	go func() {
		defer close(lineChan)
		defer f.Close()

		str := ""
		for {
			buffer := make([]byte, 8)
			bytesRead, err := f.Read(buffer)
			if err != nil {
				break
			}

			for i := 0; i < bytesRead; i++ {
				if buffer[i] == '\n' {
					lineChan <- str
					str = ""
				} else {
					str = str + string(buffer[i])
				}
			}

		}
		if str != "" {
			lineChan <- str
		}

	}()

	return lineChan
}

func main() {

	//read from from file
	// file, err := os.Open("messages.txt")

	// if err != nil {
	// 	log.Fatalf("Error while opening the file:%v", err)
	// }

	//read from tcp connection
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatalf("Error while listening for tcp: %v", err)

	}

	conn, err := listener.Accept()
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		log.Fatalf("Error while making a connection: %v", err)
	}
	defer conn.Close()

	parsedRequest, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("Error while parsing the request: %v", err)
		return
	}

	fmt.Printf("Request Line:\n- Method: %v\n- Target: %v\n- Version: %v", parsedRequest.RequestLine.Method, parsedRequest.RequestLine.RequestTarget, parsedRequest.RequestLine.HttpVersion)
	fmt.Printf("\n\nHeaders:\n")
	parsedRequest.Headers.ForEach(func(name, value string) {
		fmt.Printf("- %s: %s\n", name, value)
	})

	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Length: 2\r\n" +
		"Connection: close\r\n" +
		"\r\n" +
		"OK"

	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
	conn.Close()
}
