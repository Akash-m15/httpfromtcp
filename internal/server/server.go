package server

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Akash-m15/httpfromtcp/internal/headers"
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
func respond400() string {
	return `
	<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`
}

func toStr(bytes []byte) string {
	out := ""
	for _, b := range bytes {
		out += fmt.Sprintf("%x", b)
	}
	return out
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
	h := response.GetDefaultHeaders(0)
	trailer := headers.NewHeaders()

	r, err := request.RequestFromReader(conn)
	if err != nil {
		responseWriter.WriteStatusLine(response.StatusBadRequest)
		responseWriter.WriteHeaders(h)
		return
	}

	errorHandler := s.Handler(responseWriter, r)

	var status response.StatusCode = response.StatusOk
	var body []byte = nil

	if errorHandler != nil {
		status = errorHandler.StatusCode
		body = []byte(errorHandler.Message)
	} else {
		if strings.HasPrefix(r.RequestLine.RequestTarget, "/httpbin/") {
			target := r.RequestLine.RequestTarget
			res, err := http.Get("http://httpbin.org/" + target[len("/httpbin/"):])
			if err != nil {
				status = response.StatusBadRequest
				body = []byte(respond400())
			}

			status = response.StatusOk
			h.Delete("Content-Length")
			h.Set("Trailer", "X-Content-SHA256")
			h.Set("Trailer", "X-Content-Length")
			h.Set("Transfer-Encoding", "chunked")
			h.Replace("Content-Type", "text/plain")

			fullBody := []byte{}
			for {
				data := make([]byte, 32)
				n, err := res.Body.Read(data)
				if err != nil {
					break
				}
				fullBody = append(fullBody, data[:n]...)
				body = append(body, fmt.Appendf(nil, "%x\r\n", n)...)
				body = append(body, data[:n]...)
				body = append(body, []byte("\r\n")...)
			}
			body = append(body, []byte("0\r\n")...)
			out := sha256.Sum256(fullBody)
			trailer.Set("X-Content-SHA256", toStr(out[:]))
			trailer.Set("X-Content-Length", fmt.Sprintf("%d", len(fullBody)))

		} else if r.RequestLine.RequestTarget == "/video" {
			f, _ := os.ReadFile("assets/vim.mp4")
			h.Replace("Content-Type", "video/mp4")
			body = f
			h.Replace("Content-Length", fmt.Sprintf("%d", len(body)))

		} else {
			body = []byte(respond200())
			status = response.StatusOk

			ok := h.Replace("Content-Length", fmt.Sprintf("%d", len(body)))
			if !ok {
				h.Set("Content-Length", fmt.Sprintf("%d", len(body)))
			}
			ok = h.Replace("Content-Type", "text/html")
		}
	}

	responseWriter.WriteStatusLine(status)
	responseWriter.WriteHeaders(h)
	responseWriter.WriteBody(body)
	responseWriter.WriteHeaders(trailer)

	// _, err = conn.Write(body)
	// if err != nil {
	// 	fmt.Printf("Write error: %v", err)
	// 	return
	// }

	time.Sleep(50 * time.Millisecond)
}
