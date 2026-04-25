package server

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Akash-m15/httpfromtcp/internal/request"
	"github.com/Akash-m15/httpfromtcp/internal/response"
)

type Server struct {
	Handler Handler
	Closed  bool
}

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Handler func(w *response.Writer, req *request.Request) *HandlerError

func respond200() string {
	return `<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`
}

func ServerInit(s *Server, port int) error {
	listener, err := net.Listen("tcp", "127.0.0.1:42069")
	if err != nil {
		fmt.Printf("Error while Listening: %v", err)
		return err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			// conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			if s.Closed {
				return
			}
			if err != nil {
				fmt.Printf("Error while accepting conn: %v", err)
				return
			}

			s.handle(conn)
		}
	}()

	return nil
}

func Serve(port int, handler Handler) (*Server, error) {
	server := &Server{
		Handler: handler,
		Closed:  false,
	}

	err := ServerInit(server, port)
	if err != nil {
		return nil, err // ← Return error, don't exit
	}
	return server, nil
}

func (s *Server) Close() error {
	s.Closed = true
	return nil
}

func (s *Server) listen() {

}

func (s *Server) handle(conn net.Conn) {
	go runConnection(s, conn)
}

func runConnection(s *Server, conn io.ReadWriteCloser) {
	defer conn.Close()
	// body := "Hello World!" // 12 bytes exactly

	responseWriter := response.NewWriter(conn)
	headers := response.GetDefaultHeaders(0)

	r, err := request.RequestFromReader(conn)
	if err != nil {
		responseWriter.WriteStatusLine(response.StatusBadRequest)
		responseWriter.WriteHeaders(headers)
		return
	}

	errorHandler := s.Handler(responseWriter, r)

	var status response.StatusCode = response.StatusOk
	var body []byte = nil

	if errorHandler != nil {
		status = errorHandler.StatusCode
		body = []byte(errorHandler.Message)
	} else {
		body = []byte(respond200())
		status = response.StatusOk
	}

	ok := headers.Replace("Content-Length", fmt.Sprintf("%d", len(body)))
	if !ok {
		headers.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}

	ok = headers.Replace("Content-Type", "text/html")

	responseWriter.WriteStatusLine(status)
	responseWriter.WriteHeaders(headers)
	responseWriter.WriteBody(body)

	// _, err = conn.Write(body)
	// if err != nil {
	// 	fmt.Printf("Write error: %v", err)
	// 	return
	// }

	time.Sleep(50 * time.Millisecond)
}
