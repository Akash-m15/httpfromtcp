package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParse(t *testing.T) {
	// Test: Valid single Header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hostVal, _ := headers.Get("HOST")
	assert.Equal(t, "localhost:42069", hostVal)
	hostVal, _ = headers.Get("MissingKey")
	assert.Equal(t, "", hostVal)
	assert.Equal(t, 25, n)
	assert.True(t, done)

	// Test: Valid  header
	headers = NewHeaders()
	data = []byte("Host: localhost:42069\r\nFooFoo: barbar \r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hostVal, _ = headers.Get("HOST")
	assert.Equal(t, "localhost:42069", hostVal)
	hostVal, _ = headers.Get("MissingKey")
	assert.Equal(t, "", hostVal)
	hostVal, _ = headers.Get("FooFoo")
	assert.Equal(t, "barbar", hostVal)
	assert.Equal(t, 40, n)
	assert.False(t, done)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("Host: localhost:42069\r\n Host: localhost:42068 \r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	hostVal, _ = headers.Get("HOST")
	assert.Equal(t, "localhost:42069,localhost:42068", hostVal)
	hostVal, _ = headers.Get("MissingKey")
	assert.Equal(t, "", hostVal)
	assert.Equal(t, 50, n)
	assert.True(t, done)
}
