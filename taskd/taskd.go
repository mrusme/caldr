package taskd

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
)

const (
	STATUS_SUCCESS     = 200
	STATUS_NO_CHANGE   = 201
	STATUS_DEPRECATED  = 300
	STATUS_REDIRECT    = 301
	STATUS_RETRY       = 302
	STATUS_MALFORMED   = 400
	STATUS_UNSUPPORTED = 401
	STATUS_UNAVAIL     = 420
	STATUS_SHUTDOWN    = 421
	STATUS_DENIED      = 430
	STATUS_SUSPENDED   = 431
	STATUS_TERMINATED  = 432
	STATUS_ERROR       = 500
	STATUS_ILLEGAL     = 501
	STATUS_NI          = 502
	STATUS_CPNI        = 503
	STATUS_TOO_BIG     = 504
)

type Processor func(newSyncID string, msg Message) (Message, error)

type Taskd struct {
	port      int
	certFile  string
	keyFile   string
	cert      tls.Certificate
	tlsConfig *tls.Config
	listener  net.Listener
	processor Processor
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

func New(port int, certFile string, keyFile string, proc Processor) (Taskd, error) {
	var td Taskd
	var err error

	td.port = port
	td.certFile = certFile
	td.keyFile = keyFile
	td.cert, err = tls.LoadX509KeyPair(td.certFile, td.keyFile)
	td.processor = proc
	if err != nil {
		return td, err
	}

	td.tlsConfig = &tls.Config{Certificates: []tls.Certificate{td.cert}}
	return td, nil
}

func (td *Taskd) Launch() error {
	var err error

	td.listener, err = tls.Listen("tcp", fmt.Sprintf(":%d", td.port), td.tlsConfig)
	if err != nil {
		return err
	}
	defer td.listener.Close()

	for {
		fmt.Println("Waiting for connection ...")
		conn, err := td.listener.Accept()
		if err != nil {
			fmt.Print(err)
		} else {
			fmt.Println("Accepted connection, forking ...")
			go td.handle(conn)
		}
	}

	return nil
}

func (td *Taskd) handle(c net.Conn) {
	defer c.Close()

	r := bufio.NewReader(c)
	msg, err := td.readLoop(r)
	if err != nil {
		fmt.Println(err)
		return
	}

	syncID, err := uuid.NewRandom()
	if err != nil {
		// TODO: Handle
		fmt.Println(err)
		return
	}

	m, err := td.convertMessage(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%#v\n", m)

	nm, err := (td.processor)(syncID.String(), *m)
	if err != nil {
		err = td.respond(c, STATUS_ERROR, "error", m.SyncID, "")
		if err != nil {
			fmt.Println(err)
		}
		return
	}
	var payload string = ""
	for _, task := range nm.Tasks {
		payload += task.String() + "\n"
	}

	err = td.respond(c, STATUS_SUCCESS, "Ok", syncID.String(), "")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("\n\nSent response with sync ID %s\n\n", syncID.String())
}

func (td *Taskd) readLoop(r *bufio.Reader) ([]string, error) {
	var newmsg bool = true
	var msgsize int = 0
	var msglen int = 0
	var msg []string

	for {
		msgline, err := r.ReadString('\n')
		if err != nil {
			return []string{}, err
		}

		if newmsg {
			msgsize = bytesToDecimal([]byte(msgline[:4]))
			fmt.Printf("msgsize: %d\n", msgsize)
			newmsg = false
			msg = append(msg, strings.TrimSuffix(msgline[4:], "\n"))
		} else {
			msg = append(msg, strings.TrimSuffix(msgline, "\n"))
		}
		msglen += int(len(msgline))

		fmt.Printf("%q\n", msg[len(msg)-1])

		if msglen == msgsize {
			newmsg = true
			break
		}
	}

	return msg, nil
}

func (td *Taskd) respond(conn net.Conn, code int, status string, syncID string, payload string) error {
	resp := new(strings.Builder)
	_, err := resp.WriteString(fmt.Sprintf(
		"client: taskd 1.0.0\n"+
			"type: response\n"+
			"protocol: v1\n"+
			"code: %d\n"+
			"status: %s\n"+
			"\n"+
			"%s\n"+
			"%s"+
			"\n"+
			"\n", code, status, syncID, payload))
	if err != nil {
		return err
	}

	_, err = conn.Write(append(
		decimalToBytes(resp.Len()+4),
		[]byte(resp.String())...,
	))

	return err
}

func (td *Taskd) convertMessage(msg []string) (*Message, error) {
	var m *Message = new(Message)
	dataMap := make(map[string]string)

	for _, line := range msg {
		if len(line) == 0 {
			continue
		}
		if line[0] == '{' {
			task := Task{}
			err := json.Unmarshal([]byte(line), &task)
			if err != nil {
				return nil, err
			}
			m.Tasks = append(m.Tasks, task)
		} else {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				dataMap[parts[0]] = parts[1]
			}
		}
	}

	if err := mapstructure.Decode(dataMap, &m); err != nil {
		return nil, err
	}
	return m, nil
}
