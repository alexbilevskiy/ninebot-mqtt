package mqtt

import (
	"encoding/json"
	"fmt"
	proto "github.com/huin/mqtt"
	"github.com/jeffallen/mqtt"
	"log"
	"net"
)

type Client struct {
	options Options
	mqttC   *mqtt.ClientConn
}

type Options struct {
	Address  string
	Username string
	Password string
	ClientId string
	Topic    string
}

func Connect(options Options) (client *Client, err error) {
	conn, err := net.Dial("tcp", options.Address)
	if err != nil {
		return
	}
	mqttC := mqtt.NewClientConn(conn)
	mqttC.ClientId = options.ClientId
	err = mqttC.Connect(options.Username, options.Password)
	if err != nil {
		return
	}
	client = &Client{
		options: options,
		mqttC:   mqttC,
	}
	return
}

func (c *Client) SendFullInfo(id string, info interface{}) {
	jsonInfo, err := json.Marshal(info)
	if err != nil {
		log.Printf("json marshal error: %s", err.Error())
		return
	}

	c.mqttC.Publish(&proto.Publish{
		TopicName: fmt.Sprintf(c.options.Topic, id),
		Payload:   proto.BytesPayload(jsonInfo),
	})
}
