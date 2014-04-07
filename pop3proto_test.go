package pop3

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	// This function calls textproto.Reader.ReadLine only.
	// so no need　for test.
}

func TestReadLines(t *testing.T) {
	r := reader("line1\nline2\n.\n")

	s, err := r.ReadLines()

	if len(s) != 2 || s[0] != "line1" || s[1] != "line2" || err != nil {
		t.Fatalf("%v, %v", s, err)
	}

	s, err = r.ReadLines()

	if s != nil || err != io.EOF {
		t.Fatalf("EOF: %s, %v", s, err)
	}
}

func TestReadToPeriod(t *testing.T) {
	r := reader("line1\nline2\n.\n")

	s, err := r.ReadToPeriod()

	expected := strings.Replace(`line1
line2`, "\n", "\r\n", -1)

	if s != expected || err != nil {
		t.Fatalf("%v, %v", s, err)
	}

	s, err = r.ReadToPeriod()

	if s != "" || err != io.EOF {
		t.Fatalf("EOF: %s, %v", s, err)
	}
}

func TestReadResponse(t *testing.T) {
	var r *Reader
	var s string
	var err error

	r = reader("+OK message\n")
	s, err = r.ReadResponse()
	if s != "message" || err != nil {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("+OK\n")
	s, err = r.ReadResponse()
	if s != "" || err != nil {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("+OK \n")
	s, err = r.ReadResponse()
	if s != "" || err != nil {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("+OKAY\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "unknown response: +OKAY" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("-ERR message\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "message" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("-ERR\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("-ERR \n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("-ERROR\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "unknown response: -ERROR" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("message\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "unknown response: message" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("* message\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "unknown response: * message" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader(" message\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "unknown response:  message" {
		t.Fatalf("%v, %v", s, err)
	}

	r = reader("\n")
	s, err = r.ReadResponse()
	if s != "" || err == nil || err.Error() != "unknown response: " {
		t.Fatalf("%v, %v", s, err)
	}
}

func TestWriteLine(t *testing.T) {
	var buf bytes.Buffer

	w := NewWriter(bufio.NewWriter(&buf))

	err := w.WriteLine("foo %d", 123)

	if s := buf.String(); s != "foo 123\r\n" || err != nil {
		t.Fatalf("s=%q; err=%s", s, err)
	}
}

func reader(s string) *Reader {
	return NewReader(bufio.NewReader(strings.NewReader(s)))
}
