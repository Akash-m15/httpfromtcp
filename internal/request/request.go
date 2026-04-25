package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/Akash-m15/httpfromtcp/internal/headers"
)

const BUFFER_SIZE = 1024

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	State       ParseState
}

type ParseState int

const (
	StateInit ParseState = iota
	StateInitialized
	StateHeaders
	StateBody
	StateDone
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func newRequest() *Request {
	return &Request{
		State:   StateInitialized,
		Headers: headers.NewHeaders(),
		Body:    []byte{},
	}
}

func getInt(headers headers.Headers, name string, defaultValue int) int {
	valueStr, ok := headers.Get(name)
	if !ok {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	var parsedReq = newRequest()
	buffer := make([]byte, BUFFER_SIZE)
	bufIdx := 0
	fmt.Println("Parse start")
	for parsedReq.State != StateDone {
		fmt.Println("Inside For", string(buffer))
		if len(buffer) <= bufIdx {
			newBuffer := make([]byte, 2*len(buffer))
			copy(newBuffer, buffer[:bufIdx])
			buffer = newBuffer
		}

		//read into the buffer
		bytesRead, err := reader.Read(buffer[bufIdx:])
		fmt.Println("After Read", bytesRead, err)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("Hitting EOF", bufIdx, parsedReq.State)

				// EOF hit: try to parse whatever is left in the buffer one last time
				for bufIdx >= 0 {
					// If parsing completed the request, we're good
					if parsedReq.State == StateDone {
						return parsedReq, nil
					}
					parsed, parseErr := parsedReq.parse(buffer[:bufIdx])
					if parseErr != nil {
						return nil, parseErr
					}
					// Shift any unparsed data (usually 0 if parse succeeded)
					if parsed > 0 {
						copy(buffer, buffer[parsed:bufIdx])
						bufIdx -= parsed
					}
				}
				// Otherwise, the request was incomplete when connection closed
			}
			return nil, err
		}
		bufIdx += bytesRead

		//parse from the buffer
		bytesParsed, err := parsedReq.parse(buffer[:bufIdx])

		if err != nil {
			return nil, err
		}

		copy(buffer, buffer[bytesParsed:bufIdx])
		bufIdx -= bytesParsed

	}
	fmt.Println("Parse end")
	fmt.Println("--------------")
	return parsedReq, nil
}

func (r *Request) parse(data []byte) (int, error) {
	totalParsed := 0

	for r.State != StateDone && totalParsed < len(data) {
		switch r.State {
		case StateInitialized:
			req, n, err := parseRequestLine(data[totalParsed:])
			if err != nil {
				return 0, err
			}
			if n == 0 {
				// Cannot parse request line yet, need more data
				return totalParsed, nil
			}
			r.RequestLine = req.RequestLine
			totalParsed += n
			r.State = StateHeaders
			// Fall through to parse headers immediately

		case StateHeaders:
			n, done, err := r.Headers.Parse(data[totalParsed:])
			if err != nil {
				return 0, err
			}
			totalParsed += n
			if done {
				// Check if this request has a body
				method := r.RequestLine.Method
				hasBody := method != "GET" && method != "HEAD" && method != "DELETE"
				contentLen := getInt(r.Headers, "content-length", 0)

				if !hasBody || contentLen == 0 {
					r.State = StateDone
				} else {
					r.State = StateBody
				}
			}
			// If headers not done, we need more data → break and return
			if !done {
				return totalParsed, nil
			}
			// If done, continue loop to handle StateBody in same call

		case StateBody:
			length := getInt(r.Headers, "content-length", 0)
			if length == 0 {
				r.State = StateDone
				return totalParsed, nil
			}

			remaining := length - len(r.Body)
			available := len(data) - totalParsed
			toRead := min(remaining, available)

			r.Body = append(r.Body, data[totalParsed:totalParsed+toRead]...)
			totalParsed += toRead

			if len(r.Body) == length {
				r.State = StateDone
			}
			// If body not complete, we need more data → break
			if r.State != StateDone {
				return totalParsed, nil
			}

		case StateDone:
			return totalParsed, nil
		}
	}

	return totalParsed, nil
}

func parseRequestLine(req []byte) (*Request, int, error) {
	idx := bytes.Index(req, []byte("\r\n"))
	if idx == -1 {
		return nil, 0, nil
	}
	httpReq := strings.Split(string(req), "\r\n")
	httpReqLine := strings.Split(httpReq[0], " ")

	if len(httpReqLine) != 3 {
		return nil, 0, fmt.Errorf("Invalid")
	}

	if !IsLetter(httpReqLine[0]) {
		return nil, 0, fmt.Errorf("Method contains non alphabetic characters")
	}

	httpVersion := strings.Split(httpReqLine[2], "/")
	version := httpVersion[1]

	if version != "1.1" {
		return nil, 0, fmt.Errorf("Version does not match")
	}

	reqLine := Request{
		RequestLine: RequestLine{
			Method:        httpReqLine[0],
			RequestTarget: httpReqLine[1],
			HttpVersion:   version,
		},
	}
	return &reqLine, idx + 2, nil
}

func IsLetter(s string) bool {
	for i, r := range s {
		if i == 0 && r == '/' {
			continue
		}
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
