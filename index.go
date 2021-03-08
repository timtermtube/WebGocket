package WebGocket

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	//"net/http"
	//"strings"
)

type Cert struct {
	certFilePath string
	keyFilePath  string
}

type Send func(string)

// Client is...
type Client struct {
	Send       Send
	NetClient  net.Conn
	NetAddress net.Addr
	Id         int
}

type Eventer func(Client, string)

var (
	sHand = func(a string) string {
		a = string(a)
		return fmt.Sprintf("HTTP/1.1 101 Switching Protocols\r\nUpgrade: WebSocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", a)
	}
	fHand   = "HTTP/1.1 426 Upgrade Required\r\nUpgrade: WebSocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nContent-Type: text/plain\r\n\r\nThis service requires use of the WebSocket protocol\r\n"
	uuidKey = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	iden    = 0
)

var Users []Client

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

func reByte(data string) []byte {
	beReturned := []byte(data)
	for k, _ := range beReturned {
		if beReturned[k] == 0 {
			beReturned[k] = 129
			break
		}
	}
	return beReturned
}

func doMasking(data string) []byte {
	rebyted := reByte(data)
	opcode := byte(129)
	l := 0
	for k, _ := range rebyted {
		if rebyted[k] == 0 {
			l = k - 1
			break
		}
	}
	newData := make([]byte, 65535)
	newData[0] = opcode
	payloadLength := l
	dataLocation := 0
	if payloadLength <= 125 {
		dataLocation = 2
		newData[1] = byte(payloadLength)
	} else if payloadLength >= 126 && payloadLength <= 65535 {
		dataLocation = 4
		newData[1] = 126
		newData[2] = byte(payloadLength >> 8)
		newData[3] = byte(payloadLength & 255)
	} else if payloadLength >= 65536 {
		dataLocation = 10
		newData[1] = 127

		// The first 32 bits
		nine := payloadLength & 255
		eight := payloadLength >> 8 & 255
		seven := payloadLength >> 16 & 255
		six := payloadLength >> 24 & 255
		newData[9] = byte(nine)
		newData[8] = byte(eight)
		newData[7] = byte(seven)
		newData[6] = byte(six)

		// if the number is greater than 32bit
		if payloadLength >= 4294967296 {
			// Get the higher 64 bit
			sixtyFourBit := strconv.FormatInt(int64(payloadLength), 2)
			thirtyTwoBit := sixtyFourBit[0 : len(sixtyFourBit)-32]
			secondThirtyTwoBits, _ := strconv.ParseInt(thirtyTwoBit, 0, 2)

			newData[5] = byte((secondThirtyTwoBits) & 255)
			newData[4] = byte((secondThirtyTwoBits >> 8) & 255)
			newData[3] = byte((secondThirtyTwoBits >> 16) & 255)
			newData[2] = byte((secondThirtyTwoBits >> 24) & 255)
		} else { // if number is less than 32bit
			newData[5] = 0
			newData[4] = 0
			newData[3] = 0
			newData[2] = 0
		}
	}

	for i := 0; i < payloadLength; i++ {
		newData[dataLocation] = rebyted[i]
		dataLocation++
	}
	//fmt.Println(newData)
	return []byte(newData)
}

func handShaker(conn net.Conn, err error, path string, address string, open Eventer, message Eventer, close Eventer, wsc Client) {
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
						go close(wsc, "ClosedFinely")
						conn.Close()
						return
					}
				} else if _insideData[0] == 0 {
					if _insideData[1] == 0 {
						// Unexpected Closing
						go close(wsc, "ClosedUnexpectedly")
						conn.Close()
						return
					}
				} else {
					// Success to Read
					data := string(decoded)
					//fmt.Println(_insideData)
					if strings.Index(data, "�") == -1 {
						fmt.Println(_insideData)
						go message(wsc, data)
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
		cxC := Client{func(data string) {
			d := doMasking(data)
			//fmt.Println(d)
			client.Write(d)
		}, client, client.RemoteAddr(), iden}
		Users = append(Users, cxC)
		iden++
		go open(cxC, "Connection")
		if err != nil {
			fmt.Println("Error:", err)
		}
		defer client.Close()
		go handShaker(client, err, path, address, open, message, close, cxC)
	}
}
