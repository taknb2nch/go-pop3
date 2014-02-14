package pop3

import (
	"bufio"
	"fmt"
	"io"
	_ "log"
	"net/textproto"
	"strings"
)

type ResponseError string

func (r ResponseError) Error() string {
	return string(r)
}

type Conn struct {
	Reader
	Writer
	conn io.ReadWriteCloser
}

func NewConn(conn io.ReadWriteCloser) *Conn {
	return &Conn{
		Reader: Reader{R: textproto.NewReader(bufio.NewReader(conn))},
		Writer: Writer{W: bufio.NewWriter(conn)},
		conn:   conn,
	}
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

type Reader struct {
	R *textproto.Reader
}

func (r *Reader) ReadLine() (string, error) {
	return r.R.ReadLine()
	// for debug
	// l, err := r.R.ReadLine()
	// log.Printf("> %s\n", l)
	// return l, err
}

func (r *Reader) ReadLines() ([]string, error) {
	var lines []string
	var line string
	var err error

	for {
		line, err = r.R.ReadLine()

		if err != nil {
			return nil, err
		}

		if line == "." {
			return lines, nil
		}

		lines = append(lines, line)
	}
}

func (r *Reader) ReadToPeriod() (string, error) {
	lines, err := r.ReadLines()

	if err != nil {
		return "", nil
	}

	return strings.Join(lines, "\r\n"), nil
}

func (r *Reader) ReadResponse() (string, error) {
	line, err := r.ReadLine()

	if err != nil {
		return "", err
	}

	return r.parseResponse(line)
}

func (r *Reader) parseResponse(line string) (string, error) {
	var index int

	if index = strings.Index(line, " "); index < 0 {
		return "", ResponseError(fmt.Sprintf("レスポンスのフォーマットが不正です。: %s", line))
	}

	switch strings.ToUpper(line[:index]) {
	case "+OK":
		return line[index+1:], nil
	case "-ERR":
		return "", ResponseError(line[index+1:])
	default:
		return "", ResponseError(fmt.Sprintf("レスポンスの内容が不明です。: %s", line))
	}
}

var crnl = []byte{'\r', '\n'}

type Writer struct {
	W *bufio.Writer
}

func (w *Writer) WriteLine(format string, args ...interface{}) error {
	var err error

	if _, err = fmt.Fprintf(w.W, format, args...); err != nil {
		return err
	}

	if _, err = w.W.Write(crnl); err != nil {
		return err
	}

	return w.W.Flush()

	// for debug
	// var err error

	// l := fmt.Sprintf(format, args...)

	// if _, err = fmt.Fprint(w.W, l); err != nil {
	// 	return err
	// }

	// if _, err = w.W.Write(crnl); err != nil {
	// 	return err
	// }

	// if err = w.W.Flush(); err != nil {
	// 	return err
	// }

	// log.Printf("< %s\n", l)

	// return nil
}
