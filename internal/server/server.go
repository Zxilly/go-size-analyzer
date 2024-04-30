package server

import (
	"fmt"
	"github.com/Zxilly/go-size-analyzer/internal/utils"
	"net"
)

const template = "" +
	"HTTP/1.1 200 OK\r\n" +
	"Content-Type: text/html\r\n" +
	"Content-Length: %d\r\n" +
	"Conection: close\r\n" +
	"Server: go-size-analyzer\r\n" +
	"Cache-Control: no-cache, no-store, must-revalidate\r\n" +
	"\r\n"

func handleConn(conn net.Conn, content []byte) {
	raw := fmt.Sprintf(template, len(content)) + string(content)
	_, _ = conn.Write([]byte(raw))
	return
}

func HostServer(content []byte, listen string) net.Listener {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		utils.FatalError(err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go handleConn(conn, content)
		}
	}()

	return l
}
