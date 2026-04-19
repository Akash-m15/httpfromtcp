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
		if len(buffer) <= bufIdx {
			newBuffer := make([]byte, 2*len(buffer))
			copy(newBuffer, buffer[:bufIdx])
			buffer = newBuffer
		}

		//read into the buffer
		bytesRead, err := reader.Read(buffer[bufIdx:])
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
	fmt.Println("Inside Parse", string(data))
	switch r.State {
	case StateInitialized:
		req, bytesParsed, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if bytesParsed == 0 {
			return 0, nil
		}
		r.RequestLine = req.RequestLine
		r.State = StateHeaders
		fmt.Println("Returning from StateInit: ", bytesParsed)
		return bytesParsed, nil

	case StateHeaders:
		bytesRead, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.State = StateBody
			fmt.Println("Returning from StateHeaders (2): ", bytesRead)
			return bytesRead, nil
		} else {
			fmt.Println("Returning from StateHeaders (1): ", bytesRead)
			return bytesRead, nil
		}

	case StateBody:
		fmt.Println("Enter StateBody")
		length := getInt(r.Headers, "content-length", 0)
		if length == 0 {
			r.State = StateDone
			fmt.Println("Returning from StateBody: (length = 0) ")
			return 0, nil
		}

		remaining := min(length-len(r.Body), len(data))
		r.Body = append(r.Body, data[:remaining]...)

		if len(r.Body) == length {
			r.State = StateDone
			fmt.Println("Returning from StateBody: (length = body)", remaining)
			return remaining, nil
		}
		if len(r.Body) > length {
			return 0, fmt.Errorf("Body greater than content length")
		}
		fmt.Println("Returning from StateBody: (1) ", remaining)
		return remaining, nil
	case StateDone:
		fmt.Println("Entering StateDone in Parse func")
		return 0, fmt.Errorf("error: trying to read data  in a done state")
	}

	return 0, fmt.Errorf("error: Unknown State")
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
