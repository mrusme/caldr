package taskd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"math"
	"net"
)

type Taskd struct {
	tlsConfig *tls.Config
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
			go func(c net.Conn) {
				defer c.Close()
				var newmsg bool = true
				var msgsize int64 = 0
				var msglen int64 = 0

				r := bufio.NewReader(conn)
				for {
					msg, err := r.ReadString('\n')
					if err != nil {
						fmt.Println(err)
						return
					}

					if newmsg {
						msgsize = int64(msg[0])*int64(math.Pow(256, 3)) +
							int64(msg[1])*int64(math.Pow(256, 2)) +
							int64(msg[2])*int64(math.Pow(256, 1)) +
							int64(msg[3])*int64(math.Pow(256, 0))
						fmt.Printf("msgsize: %d\n", msgsize)
						newmsg = false
					}
					fmt.Printf("%q", msg)
					msglen += int64(len(msg))

					if msglen == msgsize {
						break
					}
				}

				// var bla []byte
				// fmt.Println("Forked! Replying and listening ...")
				// n, err := c.Read(bla)
				// fmt.Print(n)
				// fmt.Print(err)
				//
				// buf := new(strings.Builder)
				// _, err = io.WriteString(c, "n\ntest\n")
				// if err != nil {
				// 	fmt.Print(err)
				// }
				// n2, err := io.Copy(buf, c)
				// fmt.Printf("%d\n", n2)
				// if err != nil {
				// 	fmt.Print(err)
				// } else {
				// 	fmt.Println(buf.String())
				// }
			}(conn)
		}
	}

	return t, nil
}
