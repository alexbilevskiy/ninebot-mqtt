package scooter

import (
	"github.com/aprosvetova/ninebot-mqtt/scooter/protocol"
	"github.com/aprosvetova/ninebot-mqtt/serial"
	"log"
)

func Request(req []byte) (*protocol.Response, error) {
	var parsed *protocol.Response
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

			return nil, parseErr
		}
		log.Fatalf("parse error: %s", parseErr.Error())
	}
	//printBytes("resp parsed", parsed.Payload)

	return parsed, nil

}
