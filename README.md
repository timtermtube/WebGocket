# NOTICE
It is outdated project.

# WebGocket
"WebGocket" is the simplest websocket server in Golang. It is revolution, and new inspiration of websocket.

# Documentation
TEST BLANKS

# How to run?
These are only codes what you need to run.
```golang
package main

import (
	"fmt"
	"net"

	WebGocket "github.com/timtermtube/WebGocket" // It must be installed or in go.mod as local module to import!
)

func open(client net.Conn, eventDescription string) {
	fmt.Println(eventDescription)
	/*...*/
}

func message(client net.Conn, message string) {
	fmt.Println(message)
	/*...*/
}

func close(client net.Conn, guessedReason string) {
	fmt.Println(guessedReason)
	/*...*/
}

func main() {
	WebGocket.ServerOpen("/", ":8084", open, message, close)
}

```
