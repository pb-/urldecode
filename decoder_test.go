package urldecode

import (
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func readAllBytewise(t *testing.T, r io.Reader) string {
	var data []byte
	b := []byte{0}

	for {
		n, err := r.Read(b)
		if n > 0 {
			data = append(data, b...)
		}
		if err == io.EOF {
			break
		}
		require.Nil(t, err)
	}

	return string(data)
}

func TestBytewise(t *testing.T) {
	r := bytes.NewBufferString("abcdef")
	require.Equal(t, "abcdef", readAllBytewise(t, r))
}

func readAll(t *testing.T, r io.Reader) string {
	data, err := ioutil.ReadAll(r)
	require.Nil(t, err)
	return string(data)
}

func requireKeyValue(t *testing.T, expectedKey, expectedValue string, decoder *Decoder) {
	key, value, err := decoder.NextPair()
	require.Nil(t, err)
	require.Equal(t, expectedKey, key, "key does not match")
	require.NotNil(t, value, "non-nil value expected")
	require.Equal(t, expectedValue, readAll(t, value), "value does not match")
}

func requireEOF(t *testing.T, decoder *Decoder) {
	_, _, err := decoder.NextPair()
	require.Equal(t, io.EOF, err)
}

func TestBasic(t *testing.T) {
	d := NewDecoder(bytes.NewBufferString(""))
	key, value, err := d.NextPair()
	require.Equal(t, io.EOF, err)
	require.Equal(t, "", key)
	require.Nil(t, value)

	d = NewDecoder(bytes.NewBufferString("foo"))
	requireKeyValue(t, "foo", "", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=bar"))
	requireKeyValue(t, "foo", "bar", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=b+r"))
	requireKeyValue(t, "foo", "b r", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%"))
	requireKeyValue(t, "foo", "%", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%a"))
	requireKeyValue(t, "foo", "%a", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%aa"))
	requireKeyValue(t, "foo", "\xaa", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%20"))
	requireKeyValue(t, "foo", " ", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%r"))
	requireKeyValue(t, "foo", "%r", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%ar"))
	requireKeyValue(t, "foo", "%ar", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%00%01%fe%FF"))
	requireKeyValue(t, "foo", "\x00\x01\xfe\xff", d)
	requireEOF(t, d)
}

func TestMany(t *testing.T) {
	d := NewDecoder(bytes.NewBufferString("foo=bar&baz=qux"))
	requireKeyValue(t, "foo", "bar", d)
	requireKeyValue(t, "baz", "qux", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=bar&baz=%"))
	requireKeyValue(t, "foo", "bar", d)
	requireKeyValue(t, "baz", "%", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo&bar&baz"))
	requireKeyValue(t, "foo", "", d)
	requireKeyValue(t, "bar", "", d)
	requireKeyValue(t, "baz", "", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=&bar="))
	requireKeyValue(t, "foo", "", d)
	requireKeyValue(t, "bar", "", d)
	requireEOF(t, d)

	d = NewDecoder(bytes.NewBufferString("foo=%&bar="))
	requireKeyValue(t, "foo", "%", d)
	requireKeyValue(t, "bar", "", d)
	requireEOF(t, d)
}

func TestBytewiseDecoding(t *testing.T) {
	d := NewDecoder(bytes.NewBufferString("foo=ba%d0r&baz=qux%0"))

	key, value, err := d.NextPair()
	require.Nil(t, err)
	require.Equal(t, "foo", key)
	require.NotNil(t, value)
	require.Equal(t, "ba\xd0r", readAllBytewise(t, value))

	key, value, err = d.NextPair()
	require.Nil(t, err)
	require.Equal(t, "baz", key)
	require.NotNil(t, value)
	require.Equal(t, "qux%0", readAllBytewise(t, value))

	requireEOF(t, d)
}

func TestRefill(t *testing.T) {
	for shift := 0; shift < 5; shift++ {
		key1 := []byte{'a', '='}
		value1 := bytes.Repeat([]byte{'b'}, bufferSize-len(key1)-shift)
		keyvalue2 := []byte{'&', 'c', '=', 'd'}
		d := NewDecoder(bytes.NewBuffer(append(append(key1, value1...), keyvalue2...)))
		requireKeyValue(t, "a", string(value1), d)
		requireKeyValue(t, "c", "d", d)
		requireEOF(t, d)
	}
}

func TestDecoding(t *testing.T) {
	key := []byte{'a', '='}
	value := []byte{}
	for i := 0; i < 2; i++ { // make sure we exceed bufferSize
		for j := 0; j < 256; j++ {
			value = append(value, byte(j))
		}
	}
	valueEncoded := make([]byte, len(value)*3)
	for i := 0; i < len(value); i++ {
		valueEncoded[i*3] = '%'
		hex.Encode(valueEncoded[i*3+1:i*3+3], value[i:i+1])
	}

	d := NewDecoder(bytes.NewBuffer(append(key, valueEncoded...)))
	requireKeyValue(t, "a", string(value), d)
	requireEOF(t, d)
}

func TestKeyTooLong(t *testing.T) {
	_, _, err := NewDecoder(bytes.NewBuffer(bytes.Repeat([]byte{0}, bufferSize))).NextPair()
	require.Error(t, err)
}
