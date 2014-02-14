// Package pop3 implements the Post Office Protocol version 3.
package pop3

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// MessageInfoはメッセージ番号、メッセージサイズ、およびユニークIDを持つ型です。
// ListAllまたはUidlAllメソッドの戻り値として使用されます。
// ListAllメソッドにて取得した場合は、メッセージ番号とメッセージサイズ、
// UidlAllメソッドにて取得した場合は、メッセージ番号とユニークIDのみに値が入っています。
type MessageInfo struct {
	Number int
	Size   uint64
	Uid    string
}

// ClientはPOPサーバーへの接続を表すクライアントです。
type Client struct {
	// TextはClientによって使用されるmail2.Connです。
	Text *Conn
	// keep a reference to the connection so it can be used to create a TLS
	// connection later
	conn net.Conn
}

// Dialは指定されたアドレスのPOPサーバーに接続された新規Clientを返します。
// アドレスはポート番号を含まなくてはなりません。
func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return NewClient(conn)
}

// NewClient既存のコネクションを使用した新規Clientを返します。
func NewClient(conn net.Conn) (*Client, error) {
	text := NewConn(conn)

	_, err := text.ReadResponse()

	if err != nil {
		return nil, err
	}

	return &Client{Text: text, conn: conn}, nil
}

// Userは指定されたユーザを使用してサーバーに対してUSERコマンドを発行します。
func (c *Client) User(user string) error {
	return c.cmdSimple("USER %s", user)
}

// Passは指定されたパスワードを使用してサーバーに対してPASSコマンドを発行します。
func (c *Client) Pass(pass string) error {
	return c.cmdSimple("PASS %s", pass)
}

// Statはサーバーに対してSTARコマンドを発行します。
// サーバーに保存されているメール数、合計メールサイズおよびエラーが返ります。
func (c *Client) Stat() (int, uint64, error) {
	return c.cmdStatOrList("STAT", "STAT")
}

// Retrは指定されたメール番号を使用してサーバーに対してRetrコマンドを発行します。
// サーバーに保存されているメール数、合計メールサイズが返ります。
func (c *Client) Retr(number int) (string, error) {
	var err error

	err = c.Text.WriteLine("RETR %d", number)

	if err != nil {
		return "", err
	}

	_, err = c.Text.ReadResponse()

	if err != nil {
		return "", err
	}

	return c.Text.ReadToPeriod()
}

// Listは指定されたメール番号を使用してサーバーに対してLISTコマンドを発行します。
// 指定されたメール番号が存在する場合は、メール番号とサイズが返ります。
func (c *Client) List(number int) (int, uint64, error) {
	return c.cmdStatOrList("LIST", "LIST %d", number)
}

// ListAllはサーバーに対してLISTコマンドを発行します。
// 存在するメール件数分のMessageInfoが返ります。
func (c *Client) ListAll() ([]MessageInfo, error) {
	list := make([]MessageInfo, 0)

	err := c.cmdReadLines("LIST", func(line string) error {
		number, size, err := c.convertNumberAndSize(line)

		if err != nil {
			return err
		}

		list = append(list, MessageInfo{Number: number, Size: size})

		return nil
	})

	if err != nil {
		return nil, ResponseError(fmt.Sprintf("LISTコマンドのレスポンスが不正です。: %s", err.Error()))
	}

	return list, nil
}

// Uidlは指定されたメール番号を使用してサーバーに対してUIDLコマンドを発行します。
// 指定されたメール番号が存在する場合は、メール番号とユニークIDが返ります。
func (c *Client) Uidl(number int) (int, string, error) {
	var err error

	err = c.Text.WriteLine("UIDL %d", number)

	if err != nil {
		return 0, "", err
	}

	var msg string

	msg, err = c.Text.ReadResponse()

	if err != nil {
		return 0, "", err
	}

	var val int
	var uid string

	val, uid, err = c.convertNumberAndUid(msg)

	if err != nil {
		return 0, "", ResponseError(fmt.Sprintf("UIDLコマンドのレスポンスが不正です。: %s", err.Error()))
	}

	return val, uid, nil
}

// UidlAllはサーバーに対してUIDLコマンドを発行します。
// 存在するメール件数分のMessageInfoが返ります。
func (c *Client) UidlAll() ([]MessageInfo, error) {
	list := make([]MessageInfo, 0)

	err := c.cmdReadLines("UIDL", func(line string) error {
		number, uid, err := c.convertNumberAndUid(line)

		if err != nil {
			return err
		}

		list = append(list, MessageInfo{Number: number, Uid: uid})

		return nil
	})

	if err != nil {
		return nil, ResponseError(fmt.Sprintf("UIDLコマンドのレスポンスが不正です。: %s", err.Error()))
	}

	return list, nil
}

// Deleは指定されたメール番号を使用してサーバーに対してDELEコマンドを発行します。
func (c *Client) Dele(number int) error {
	return c.cmdSimple("DELE %d", number)
}

// Noopはサーバーに対してNOOPコマンドを発行します。
func (c *Client) Noop() error {
	return c.cmdSimple("NOOP")
}

// Rsetはサーバーに対してRSETコマンドを発行します。
func (c *Client) Rset() error {
	return c.cmdSimple("RSET")
}

// Quitはサーバーに対してQUITコマンドを発行します。
func (c *Client) Quit() error {
	return c.cmdSimple("QUIT")
}

func (c *Client) cmdSimple(format string, args ...interface{}) error {
	var err error

	err = c.Text.WriteLine(format, args...)

	if err != nil {
		return err
	}

	_, err = c.Text.ReadResponse()

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) cmdStatOrList(name, format string, args ...interface{}) (int, uint64, error) {
	var err error

	err = c.Text.WriteLine(format, args...)

	if err != nil {
		return 0, 0, err
	}

	var msg string

	msg, err = c.Text.ReadResponse()

	if err != nil {
		return 0, 0, err
	}

	s := strings.Split(msg, " ")

	if len(s) < 2 {
		return 0, 0, ResponseError(fmt.Sprintf("%sコマンドのレスポンスが不正です。: %s", name, msg))
	}

	var val int
	var size uint64

	val, size, err = c.convertNumberAndSize(msg)

	if err != nil {
		return 0, 0, ResponseError(fmt.Sprintf("%sコマンドのレスポンスが不正です。: %s", name, err.Error()))
	}

	return val, size, nil
}

func (c *Client) cmdReadLines(cmnd string, lineFn lineFunc) error {
	var err error

	err = c.Text.WriteLine(cmnd)

	if err != nil {
		return err
	}

	_, err = c.Text.ReadResponse()

	if err != nil {
		return err
	}

	var lines []string

	lines, err = c.Text.ReadLines()

	if err != nil {
		return err
	}

	for _, line := range lines {
		err = lineFn(line)

		if err != nil {
			return err
		}
	}

	return nil
}

type lineFunc func(line string) error

func (c *Client) Close() error {
	return c.Text.Close()
}

func (c *Client) convertNumberAndSize(line string) (int, uint64, error) {
	var err error

	s := strings.Split(line, " ")

	if len(s) < 2 {
		return 0, 0, errors.New(fmt.Sprintf("分割後の配列数が2未満です。: %s", line))
	}

	var val int
	var size uint64

	if val, err = strconv.Atoi(s[0]); err != nil {
		return 0, 0, errors.New(fmt.Sprintf("配列要素[0]をint型に変換できません。: %s", line))
	}

	if size, err = strconv.ParseUint(s[1], 10, 64); err != nil {
		return 0, 0, errors.New(fmt.Sprintf("配列要素[1]をuint64型に変換できません。: %s", line))
	}

	return val, size, nil
}

func (c *Client) convertNumberAndUid(line string) (int, string, error) {
	var err error

	s := strings.Split(line, " ")

	if len(s) < 2 {
		return 0, "", errors.New(fmt.Sprintf("分割後の配列数が2未満です。: %s", line))
	}

	var val int

	if val, err = strconv.Atoi(s[0]); err != nil {
		return 0, "", errors.New(fmt.Sprintf("配列要素[0]をint型に変換できません。: %s", line))
	}

	return val, s[1], nil
}
