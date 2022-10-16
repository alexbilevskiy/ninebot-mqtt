package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
)

const ReadRegisterCommand RequestCommand = 0x01

func GetStatus() []byte {
	return CreateRequest(ReadRegisterCommand, 0x30, 0x02)
}

func GetSerialNumber() []byte {
	return CreateRequest(ReadRegisterCommand, 0x10, 0x0E)
}

func GetRemainingCapacityPerc() []byte {
	return CreateRequest(ReadRegisterCommand, 0x32, 0x02)
}

func GetRemainingCapacity() []byte {
	return CreateRequest(ReadRegisterCommand, 0x31, 0x02)
}

func GetActualCapacity() []byte {
	return CreateRequest(ReadRegisterCommand, 0x19, 0x02)
}

func GetFactoryCapacity() []byte {
	return CreateRequest(ReadRegisterCommand, 0x18, 0x02)
}

func GetCurrent() []byte {
	return CreateRequest(ReadRegisterCommand, 0x33, 0x02)
}

func GetVoltage() []byte {
	return CreateRequest(ReadRegisterCommand, 0x34, 0x02)
}

func GetTemperature() []byte {
	return CreateRequest(ReadRegisterCommand, 0x35, 0x02)
}

func CreateRequest(command RequestCommand, param byte, payload ...byte) []byte {
	cmd := []byte{0x5A, 0xA5, 0x00, 0x20, 0x22, byte(command), param}
	cmd = append(cmd, payload...)
	cmd[2] = byte(len(payload))
	cmd = append(cmd, getChecksum(cmd[2:])...)
	return cmd
}

func ParseResponse(raw []byte) (*Response, error) {
	if len(raw) < 9 {
		return nil, errors.New("raw is too short")
	}
	if raw[0] != 0x5A || raw[1] != 0xA5 {
		return nil, errors.New("not a Ninebot ES raw")
	}
	if raw[2] != uint8(len(raw)-9) {
		return nil, errors.New("wrong payload length byte")
	}
	if !bytes.Equal(raw[len(raw)-2:], getChecksum(raw[2:len(raw)-2])) {
		return nil, errors.New("wrong checksum")
	}
	response := &Response{
		Command:   raw[5],
		Parameter: raw[6],
		Payload:   raw[7 : len(raw)-2],
	}
	return response, nil
}

func getChecksum(part []byte) []byte {
	chkSum := 0xFFFF
	for _, b := range part {
		chkSum -= int(b)
	}
	bChkSum := make([]byte, 2)
	binary.LittleEndian.PutUint16(bChkSum, uint16(chkSum))
	return bChkSum
}

func ToInt16(bytesSlice []byte) int16 {
	var result int16 = 0
	buf := bytes.NewReader(bytesSlice)
	err := binary.Read(buf, binary.LittleEndian, &result)
	if err != nil {
		log.Fatalf("convert error: %s", err.Error())
	}
	return result
}
