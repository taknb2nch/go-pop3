package pop3

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSingleLineResponse(t *testing.T) {
	if len(helloServer) != len(helloClient) {
		t.Fatalf("Hello server and client size mismatch")
	}

	for i := 0; i < len(helloServer); i++ {
		execute(t, helloServer[i], helloClient[i], func(t *testing.T, c *Client) {
			var err error

			switch i {
			case 0:
				err = c.User("testuser")
			case 1:
				err = c.Pass("testpassword")
			case 2:
				var count int
				var size uint64
				count, size, err = c.Stat()

				if err == nil {
					if count != 6 {
						t.Errorf("Stat count:\nexpected:\n[%d]\nactual:\n[%d]\n", 6, count)
					}
					if size != 78680 {
						t.Errorf("Stat size:\nexpected:\n[%d]\nactual:\n[%d]\n", 78680, count)
					}
				}
			case 3:
				//_, err = c.Retr(1)
			case 4:
				var number int
				var size uint64
				number, size, err = c.List(1)

				if err == nil {
					if number != 1 {
						t.Errorf("List number:\nexpected:\n[%d]\nactual:\n[%d]\n", 1, number)
					}
					if size != 4404 {
						t.Errorf("List size:\nexpected:\n[%d]\nactual:\n[%d]\n", 4404, size)
					}
				}
			case 5:
				//_, err = c.ListAll()
			case 6:
				var number int
				var uid string
				number, uid, err = c.Uidl(1)

				if err == nil {
					if number != 1 {
						t.Errorf("Uidl number:\nexpected:\n[%d]\nactual:\n[%d]\n", 1, number)
					}
					if uid != "DJzjbtr5hb2Lefq5Ass6eMjtEBV" {
						t.Errorf("Uidl uid:\nexpected:\n[%s]\nactual:\n[%s]\n", "DJzjbtr5hb2Lefq5Ass6eMjtEBV", uid)
					}
				}
			case 7:
				//_, err = c.UidlAll()
			case 8:
				err = c.Dele(1)
			case 9:
				err = c.Noop()
			case 10:
				err = c.Rset()
			case 11:
				err = c.Quit()
			default:
				t.Fatalf("Unhandled command")
			}

			if err != nil {
				t.Errorf("Command %d failed: %v", i, err)
			}
		})
	}
}

func TestRetr(t *testing.T) {
	execute(t, retrServer, retrClient, func(t *testing.T, c *Client) {
		var err error
		var content string

		content, err = c.Retr(1)

		if err != nil {
			t.Fatalf("Command failed: %v", err)
		}

		expected := `Date: Sat, 15 Feb 2014 13:49:28 +0900 (JST)
From: <from@example.com>
Subject: test mail
To: to@example.com
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

This is a test mail.`

		expected = strings.Replace(expected, "\n", "\r\n", -1)

		if content != expected {
			t.Fatalf("excepted:\n[%s]\nactual:\n[%s]\n", expected, content)
		}
	})
}

func TestListAll(t *testing.T) {
	execute(t, listAllServer, listAllClient, func(t *testing.T, c *Client) {
		var err error

		exceptedMis := []MessageInfo{
			MessageInfo{Number: 1, Size: 4404},
			MessageInfo{Number: 2, Size: 3921},
			MessageInfo{Number: 3, Size: 6646},
			MessageInfo{Number: 4, Size: 55344},
			MessageInfo{Number: 5, Size: 5376},
			MessageInfo{Number: 6, Size: 2989},
		}

		var actualMis []MessageInfo

		actualMis, err = c.ListAll()

		if err != nil {
			t.Fatalf("Command failed: %v", err)
		}

		for i, mi := range actualMis {
			if mi.Number != exceptedMis[i].Number {
				t.Fatalf("%d: Number\nexcepted:\n[%d]\nactual:\n[%d]\n", i, exceptedMis[i].Number, mi.Number)
			}

			if mi.Size != exceptedMis[i].Size {
				t.Fatalf("%d: Size\nexcepted:\n[%d]\nactual:\n[%d]\n", i, exceptedMis[i].Size, mi.Size)
			}
		}
	})
}

func TestUidlAll(t *testing.T) {
	execute(t, uidlAllServer, uidlAllClient, func(t *testing.T, c *Client) {
		var err error

		exceptedMis := []MessageInfo{
			MessageInfo{Number: 1, Uid: "DJzjbtr5hb2Lefq5Ass6eMjtEBV"},
			MessageInfo{Number: 2, Uid: "IeHR1LtUfd5MI1gz1sYOU4TG1rx"},
			MessageInfo{Number: 3, Uid: "Gjx3lbdNAnIMCJCuQxL05DFCqyy"},
			MessageInfo{Number: 4, Uid: "DANJyGBstEYirvbBVFXSh3CGXAg"},
			MessageInfo{Number: 5, Uid: "uhNSJrPJBlcacoEbX9aXMUKC90n"},
			MessageInfo{Number: 6, Uid: "fZ88gZjY2NdYOQA6aij8dxCCifC"},
		}

		var actualMis []MessageInfo

		actualMis, err = c.UidlAll()

		if err != nil {
			t.Errorf("Command failed: %v", err)
		}

		for i, mi := range actualMis {
			if mi.Number != exceptedMis[i].Number {
				t.Fatalf("%d: Number\nexcepted:\n[%d]\nactual:\n[%d]\n", i, exceptedMis[i].Number, mi.Number)
			}

			if mi.Uid != exceptedMis[i].Uid {
				t.Fatalf("%d: Uid\nexcepted:\n[%s]\nactual:\n[%s]\n", i, exceptedMis[i].Uid, mi.Uid)
			}
		}
	})
}

func execute(t *testing.T, sServer, sClient string, processFn processFunc) {
	server := strings.Join(strings.Split(baseHelloServer+sServer, "\n"), "\r\n")
	client := strings.Join(strings.Split(baseHelloClient+sClient, "\n"), "\r\n")

	var cmdbuf bytes.Buffer

	bcmdbuf := bufio.NewWriter(&cmdbuf)

	var fake faker
	fake.ReadWriter = bufio.NewReadWriter(bufio.NewReader(strings.NewReader(server)), bcmdbuf)

	c, err := NewClient(fake)

	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	defer c.Close()

	processFn(t, c)

	bcmdbuf.Flush()

	actualcmds := cmdbuf.String()

	if client != actualcmds {
		t.Errorf("Got:\n[%s]\nExpected:\n[%s]", actualcmds, client)
	}
}

type processFunc func(t *testing.T, c *Client)

var baseHelloServer = `+OK hello from popgate(2.35.25)
`

var helloServer = []string{
	"+OK password required.\n",
	"+OK maildrop ready, 6 messages (78680 octets)\n",
	"+OK 6 78680\n",
	"+OK 4404 octets\n",
	"+OK 1 4404\n",
	"+OK 6 messages (78680 octets)\n",
	"+OK 1 DJzjbtr5hb2Lefq5Ass6eMjtEBV\n",
	"+OK 6 messages (78680 octets)\n",
	"+OK message 1 marked deleted\n",
	"+OK \n",
	"+OK maildrop has 0 messages (0 octets)\n",
	"+OK server signing off.\n",
}

var baseHelloClient = ``

var helloClient = []string{
	"USER testuser\n",
	"PASS testpassword\n",
	"STAT\n",
	"", //"RETR 1\n",
	"LIST 1\n",
	"", //"LIST\n",
	"UIDL 1\n",
	"", //"UIDL\n",
	"DELE 1\n",
	"NOOP\n",
	"RSET\n",
	"QUIT\n",
}

var retrServer = `+OK 3970 octets
Date: Sat, 15 Feb 2014 13:49:28 +0900 (JST)
From: <from@example.com>
Subject: test mail
To: to@example.com
MIME-Version: 1.0
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: quoted-printable

This is a test mail.
.
`

var retrClient = `RETR 1
`

var listAllServer = `+OK 6 messages (78680 octets)
1 4404
2 3921
3 6646
4 55344
5 5376
6 2989
.
`
var listAllClient = `LIST
`

var uidlAllServer = `+OK 6 messages (78680 octets)
1 DJzjbtr5hb2Lefq5Ass6eMjtEBV
2 IeHR1LtUfd5MI1gz1sYOU4TG1rx
3 Gjx3lbdNAnIMCJCuQxL05DFCqyy
4 DANJyGBstEYirvbBVFXSh3CGXAg
5 uhNSJrPJBlcacoEbX9aXMUKC90n
6 fZ88gZjY2NdYOQA6aij8dxCCifC
.
`
var uidlAllClient = `UIDL
`

type faker struct {
	io.ReadWriter
}

func (f faker) Close() error                     { return nil }
func (f faker) LocalAddr() net.Addr              { return nil }
func (f faker) RemoteAddr() net.Addr             { return nil }
func (f faker) SetDeadline(time.Time) error      { return nil }
func (f faker) SetReadDeadline(time.Time) error  { return nil }
func (f faker) SetWriteDeadline(time.Time) error { return nil }

func TestReceiveMail(t *testing.T) {
	t.Skipf("sorry this test has not implemented.")
}
