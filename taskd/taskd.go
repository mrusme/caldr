package taskd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"strings"

	"github.com/google/uuid"
)

type Taskd struct {
	tlsConfig *tls.Config
}

func bytesToDecimal(bytes []byte) int {
	return int(bytes[0])*int(math.Pow(256, 3)) +
		int(bytes[1])*int(math.Pow(256, 2)) +
		int(bytes[2])*int(math.Pow(256, 1)) +
		int(bytes[3])*int(math.Pow(256, 0))
}

func decimalToBytes(decimal int) []byte {
	byte1 := byte(decimal / (256 * 256 * 256))
	byte2 := byte((decimal % (256 * 256 * 256)) / (256 * 256))
	byte3 := byte(((decimal % (256 * 256 * 256)) % (256 * 256)) / 256)
	byte4 := byte(((decimal % (256 * 256 * 256)) % (256 * 256)) % 256)

	bytes := []byte{byte1, byte2, byte3, byte4}
	return bytes
}

func Launch(port int, crt string, key string) (Taskd, error) {
	var t Taskd

	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return t, err
	}

	t.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}

	l, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), t.tlsConfig)
	if err != nil {
		return t, err
	}
	defer l.Close()

	for {
		fmt.Println("Waiting for connection ...")
		conn, err := l.Accept()
		if err != nil {
			fmt.Print(err)
		} else {
			fmt.Println("Accepted connection, forking ...")
			go Handle(conn)
		}
	}

	return t, nil
}

func Handle(c net.Conn) {
	defer c.Close()

	r := bufio.NewReader(c)
	msgbuf, err := ReadLoop(r)
	if err != nil {
		fmt.Println(err)
	}

	syncId, err := uuid.NewRandom()
	if err != nil {
		// TODO: Handle
		fmt.Println(err)
	}

	// TODO: Process msgbuf

	resp := new(strings.Builder)
	resp.WriteString(
		"client: taskd 1.0.0\n" +
			"type: response\n" +
			"protocol: v1\n" +
			"code: 200\n" +
			"status: Ok\n" +
			"\n" +
			syncId.String() + "\n" +
			"\n" +
			"\n")

	_, err = c.Write(append(
		decimalToBytes(resp.Len()+4),
		[]byte(resp.String())...,
	))
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("\n\nSent response with sync ID %s\n\n", syncId.String())
}

func ReadLoop(r *bufio.Reader) ([]string, error) {
	var newmsg bool = true
	var msgsize int = 0
	var msglen int = 0
	var msgbuf []string

	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			return []string{}, err
		}

		if newmsg {
			msgsize = bytesToDecimal([]byte(msg[:4]))
			fmt.Printf("msgsize: %d\n", msgsize)
			newmsg = false
			msgbuf = append(msgbuf, strings.TrimSuffix(msg[4:], "\n"))
		} else {
			msgbuf = append(msgbuf, strings.TrimSuffix(msg, "\n"))
		}
		msglen += int(len(msg))

		fmt.Printf("%q\n", msgbuf[len(msgbuf)-1])

		if msglen == msgsize {
			newmsg = true
			break
		}
	}

	return msgbuf, nil
}
