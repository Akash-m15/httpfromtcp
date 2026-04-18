package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/Akash-m15/httpfromtcp/internal/headers"
)

const BUFFER_SIZE = 1024

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	State       ParseState
}

type ParseState int

const (
	StateInit ParseState = iota
	StateInitialized
	StateHeaders
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
	}
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	var parsedReq = newRequest()
	buffer := make([]byte, BUFFER_SIZE)
	bufIdx := 0

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
				parsedReq.State = StateDone
				break
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
	return parsedReq, nil
}

func (r *Request) parse(data []byte) (int, error) {
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
		return bytesParsed, nil

	case StateHeaders:
		bytesRead, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.State = StateDone
			return bytesRead, nil
		} else {
			return bytesRead, nil
		}

	case StateDone:
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

	fmt.Println(reqLine)
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
