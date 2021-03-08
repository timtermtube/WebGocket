package WebGocket

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	//"net/http"
	//"strings"
)

type Cert struct {
	certFilePath string
	keyFilePath  string
}

type Eventer func(net.Conn, string)

var (
	sHand = func(a string) string {
		a = string(a)
		return fmt.Sprintf("HTTP/1.1 101 Switching Protocols\r\nUpgrade: WebSocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", a)
	}
	fHand   = "HTTP/1.1 426 Upgrade Required\r\nUpgrade: WebSocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nContent-Type: text/plain\r\n\r\nThis service requires use of the WebSocket protocol\r\n"
	uuidKey = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

func HashGenerator(str string) string {
	h := sha1.New()
	h.Write([]byte(str + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	a := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return a
}

func unMasking(data []byte) []byte {
	locationMask := 2
	payloadLength := data[1] & 127

	if payloadLength == 126 {
		locationMask = 4
	} else if payloadLength == 127 {
		locationMask = 10
	}

	masking := data[locationMask : locationMask+4]
	dataLocatedAt := locationMask + 4
	decoded := make([]byte, len(data))
	x := 0

	for dataLocatedAt < len(data) {
		if data[dataLocatedAt] != 0 {
			if data[x] != 0 {
				decoded[x] = data[dataLocatedAt] ^ masking[x%4]
			}
		} else {
			break
		}
		dataLocatedAt++
		x++
	}
	return decoded
}

func doMasking(data string) bool {
	return true
}

func handShaker(conn net.Conn, err error, path string, address string, open Eventer, message Eventer, close Eventer) {
	for {
		_insideData := make([]byte, 4096)
		conn.Read(_insideData)
		data := string(_insideData)
		if err == nil {
			if strings.Index(data, "GET") == 0 {
				wsKey := ""
				hdx := strings.Split(data, "\r\n")
				for k, _ := range hdx {
					v := hdx[k]
					if strings.Index(v, "Sec-WebSocket-Key") == 0 {
						var Key string = strings.Replace(v, "Sec-WebSocket-Key: ", wsKey, 512)
						handshaked := []byte(sHand(HashGenerator(Key)))
						conn.Write(handshaked)
						break
					}
				}
			} else {
				decoded := unMasking(_insideData)
				if _insideData[0] == 136 {
					if _insideData[1] == 128 {
						// .close()
						go close(conn, "ClosedFinely")
						conn.Close()
						return
					}
				} else if _insideData[0] == 0 {
					if _insideData[1] == 0 {
						// Unexpected Closing
						go close(conn, "ClosedUnexpectedly")
						conn.Close()
						return
					}
				} else {
					// Success to Read
					data := fmt.Sprintf("%s", decoded)
					if strings.Index(data, "�") == -1 {
						go message(conn, data)
					}
				}
			}
		} else {
			return
		}
	}
}

func ServerOpen(path string, address string, open Eventer, message Eventer, close Eventer) {
	if path == "" {
		path = "/ws"
	}
	server, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	fmt.Printf("The Server is started on %s\n", address)

	for {
		client, err := server.Accept()
		go open(client, "Connection")
		if err != nil {
			fmt.Println("Error:", err)
		}
		go func() {
			go handShaker(client, err, path, address, open, message, close)
		}()
	}
}
