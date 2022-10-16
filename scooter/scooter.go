package scooter

import (
	"fmt"
	"github.com/aprosvetova/ninebot-mqtt/scooter/protocol"
	"github.com/aprosvetova/ninebot-mqtt/serial"
	"log"
)

func Request(req []byte) *protocol.Response {
	var parsed *protocol.Response
	for {
		response, requestErr := serial.Request(req)
		if requestErr != nil {
			log.Fatalf("serial request error: %s", requestErr.Error())
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
		return parsed
	}
}

func ParseCellsVoltageResp(resp []byte) map[string]int16 {
	var result = make(map[string]int16)
	var buf = make([]byte, 2)
	for k, v := range resp {
		buf[k%2] = v
		if k%2 == 1 && k != 0 {
			name := fmt.Sprintf("cell_%d", k/2)
			result[name] = protocol.ToInt16(buf)
		}
	}

	return result
}
