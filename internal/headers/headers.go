package headers

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type Headers map[string]string

var rn = []byte("\r\n")

func (h Headers) Get(name string) (string, bool) {
	value, ok := h[strings.ToLower(name)]
	return value, ok
}

func (h Headers) Set(name, value string) {
	name = strings.ToLower(name)
	v, ok := h[name]
	if ok {
		h[name] = fmt.Sprintf("%s,%s", v, value)
	} else {
		h[name] = value
	}
}

func (h Headers) Replace(name, value string) bool {
	name = strings.ToLower(name)
	_, ok := h[name]
	if ok {
		h[name] = value
	}
	return ok
}

// isTokenTable is initialized once at package load time.
var isTokenTable [256]bool

func init() {
	// Define valid tchars based on RFC 7230 / RFC 5234
	// tchar = "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." /
	//         "^" / "_" / "`" / "|" / "~" / DIGIT / ALPHA
	valid := "!#$%&'*+-.^_`|~0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(valid); i++ {
		isTokenTable[valid[i]] = true
	}
}

func (h Headers) ForEach(cb func(name, value string)) {
	for n, v := range h {
		cb(n, v)
	}
}

// IsToken checks if the key conforms to the HTTP token specification.
func IsToken(key string) bool {
	if len(key) == 0 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if !isTokenTable[key[i]] {
			return false
		}
	}
	return true
}

func NewHeaders() Headers {
	return map[string]string{} //composite literal -> returning empty map
}

func parsedFieldLine(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Malformed header")
	}

	key := parts[0]
	value := string(bytes.TrimSpace(parts[1]))

	//if key ends with whitespace, then it indicates there is space btw key and colon (Invalid format)
	if bytes.HasSuffix(key, []byte(" ")) {
		return "", "", errors.New("Invalid spacing header")
	}

	parsedKey := string(bytes.TrimSpace(key))

	if !IsToken(parsedKey) {
		return "", "", fmt.Errorf("Malformed Header name")
	}

	return parsedKey, value, nil
}

func (h Headers) Parse(data []byte) (int, bool, error) {
	read := 0
	done := false

	for {
		idx := bytes.Index(data[read:], rn)
		if idx == -1 {
			break
		}

		//return if rn found at start - indicates end of field lines
		if idx == 0 {
			read += len(rn)
			done = true
			break
		}

		key, value, err := parsedFieldLine(data[read : read+idx])
		if err != nil {
			return 0, false, err
		}

		read += idx + len(rn)
		h.Set(key, value)
	}

	return read, done, nil
}
