package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/aprosvetova/ninebot-mqtt/scooter/protocol"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	processSerial()
}

var addr string

func processSerial() {
	addr = "192.168.88.82:1234"
	checkConnection(true)
	for {
		statusReq := protocol.GetStatus()
		statusResp, err := request(statusReq)
		if err != nil {
			log.Fatalf("status request error: %s", err.Error())
		}
		status := statusResp.Payload[0]

		serialReq := protocol.GetSerial()
		serialResp, err := request(serialReq)
		if err != nil {
			log.Fatalf("serial request error: %s", err.Error())
		}
		serial := serialResp.Payload

		remainingCapacityPercReq := protocol.GetRemainingCapacityPerc()
		remainingCapacityPercResp, err := request(remainingCapacityPercReq)
		if err != nil {
			log.Fatalf("remainingCapacityPerc request error: %s", err.Error())
		}
		remainingCapacityPerc := remainingCapacityPercResp.Payload

		remainingCapacityReq := protocol.GetRemainingCapacity()
		remainingCapacityResp, err := request(remainingCapacityReq)
		if err != nil {
			log.Fatalf("remainingCapacity request error: %s", err.Error())
		}
		remainingCapacity := remainingCapacityResp.Payload

		currentReq := protocol.GetCurrent()
		currentResp, err := request(currentReq)
		if err != nil {
			log.Fatalf("current request error: %s", err.Error())
		}
		current := currentResp.Payload

		voltageReq := protocol.GetVoltage()
		voltageResp, err := request(voltageReq)
		if err != nil {
			log.Fatalf("voltage request error: %s", err.Error())
		}
		voltage := voltageResp.Payload

		temperatureReq := protocol.GetTemperature()
		temperatureResp, err := request(temperatureReq)
		if err != nil {
			log.Fatalf("temperature request error: %s", err.Error())
		}
		temperature := temperatureResp.Payload

		log.Printf(
			"[%s] [%d%% / %dmAh] status: %x; %.2f A, %.2f V; %d°/%d°",
			serial,
			toInt(remainingCapacityPerc),
			toInt(remainingCapacity),
			status,
			float64(toInt(current))*10/1000,
			float64(toInt(voltage))*10/1000,
			int(temperature[0])-20,
			int(temperature[1])-20)
	}

}

func request(req []byte) (*protocol.Response, error) {
	var parsed *protocol.Response
	for {
		checkConnection(false)
		//printBytes("status request", req)
		nBytes, errWrite := fmt.Fprintf(c, string(req))
		if errors.Is(errWrite, os.ErrDeadlineExceeded) {
			log.Printf("timeout writing to socket (%d bytes written): %s", nBytes, errWrite.Error())
			checkConnection(true)
			continue
		} else if errWrite != nil {
			log.Fatalf("error writing to socket (%d bytes written): %s", nBytes, errWrite.Error())
		}

		response, readErr := waitResponse(r)
		if errors.Is(readErr, os.ErrDeadlineExceeded) {
			log.Printf("timeout reading socket: %s", readErr.Error())
			checkConnection(true)
			continue
		} else if readErr != nil {
			log.Fatalf("error reading socket: %s", readErr.Error())
		}

		//printBytes("status response", response)
		var parseErr error
		parsed, parseErr = protocol.ParseResponse(response)
		if parseErr != nil {
			if parseErr.Error() == "wrong checksum" {
				log.Printf("parse error: %s", parseErr.Error())
				continue
			}
			log.Fatalf("parse error: %s", parseErr.Error())
		}
		//printBytes("resp parsed", parsed.Payload)

		return parsed, nil
	}
}

var c net.Conn
var r *bufio.Reader

func checkConnection(force bool) {
	var err error
	if force || c == nil {
		if c != nil {
			c.Close()
		}
		log.Printf("Connecting...")
		c, err = net.Dial("tcp", addr)
		if err != nil {
			log.Fatalf("error connecting to socket: %s", err.Error())
		}
		r = bufio.NewReader(c)
		log.Printf("Connected to %s", addr)
	}
	c.SetDeadline(time.Now().Add(2000 * time.Millisecond))
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
			return buf, nil
		}
	}
}

func waitResponseWithTimeout(reader *bufio.Reader, timeoutMs time.Duration) ([]byte, error) {
	c1 := make(chan []byte, 1)
	c2 := make(chan error, 1)
	go func() {
		bytes, err := waitResponse(reader)
		if err != nil {
			c2 <- err
		}
		c1 <- bytes
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

func printBytes(tag string, bytes []byte) {
	log.Printf("%s (%d):\t %x \t %# x", tag, len(bytes), bytes, bytes)
}

func toInt(bytes []byte) int {
	result := 0
	//for i := 0; i < len(bytes); i++ {
	//	result = result << 8
	//	result += int(bytes[i])
	//}
	//REVERSE ORDER!!!!
	for i := len(bytes) - 1; i >= 0; i-- {
		result = result << 8
		result += int(bytes[i])
	}

	return result
}
