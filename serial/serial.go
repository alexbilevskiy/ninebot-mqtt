package serial

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

var Addr string
var c net.Conn
var r *bufio.Reader

func Request(req []byte) ([]byte, error) {
	for {
		CheckConnection(false)
		//printBytes("status request", req)
		nBytes, errWrite := fmt.Fprintf(c, string(req))
		if errors.Is(errWrite, os.ErrDeadlineExceeded) {
			log.Printf("timeout writing to socket (%d bytes written): %s", nBytes, errWrite.Error())
			CheckConnection(true)
			continue
		} else if errWrite != nil {
			log.Fatalf("error writing to socket (%d bytes written): %s", nBytes, errWrite.Error())
		}

		response, readErr := waitResponse(r)
		if errors.Is(readErr, os.ErrDeadlineExceeded) {
			log.Printf("timeout reading socket: %s", readErr.Error())
			CheckConnection(true)
			continue
		} else if readErr != nil {
			log.Fatalf("error reading socket: %s", readErr.Error())
		}

		return response, nil
	}
}

func waitResponse(reader *bufio.Reader) ([]byte, error) {
	buf := make([]byte, 0)
	var awaitedLen int
	for {
		oneByte, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		buf = append(buf, oneByte)
		if len(buf) == 3 {
			awaitedLen = int(buf[2])
		} else if awaitedLen != 0 && len(buf) == awaitedLen+9 {
			//@TODO: somehow make this check abstract
			return buf, nil
		}
	}
}

func CheckConnection(force bool) {
	var err error
	if force || c == nil {
		if c != nil {
			c.Close()
		}
		log.Printf("Connecting...")
		c, err = net.Dial("tcp", Addr)
		if err != nil {
			log.Fatalf("error connecting to socket: %s", err.Error())
		}
		r = bufio.NewReader(c)
		log.Printf("Connected to %s", Addr)
	}
	c.SetDeadline(time.Now().Add(2000 * time.Millisecond))
}

func waitResponseWithTimeout(reader *bufio.Reader, timeoutMs time.Duration) ([]byte, error) {
	c1 := make(chan []byte, 1)
	c2 := make(chan error, 1)
	go func() {
		resp, err := waitResponse(reader)
		if err != nil {
			c2 <- err
		}
		c1 <- resp
	}()

	select {
	case res := <-c1:
		return res, nil
	case err := <-c2:
		return nil, err
	case <-time.After(timeoutMs * time.Millisecond):
		return nil, errors.New("timeout")
	}
}

func PrintBytes(tag string, bytes []byte) {
	log.Printf("%s (%d):\t %x \t %# x", tag, len(bytes), bytes, bytes)
}
