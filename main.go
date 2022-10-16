package main

import (
	"fmt"
	"github.com/aprosvetova/ninebot-mqtt/mqtt"
	"github.com/aprosvetova/ninebot-mqtt/scooter"
	"github.com/aprosvetova/ninebot-mqtt/scooter/protocol"
	"github.com/aprosvetova/ninebot-mqtt/serial"
	"github.com/montanaflynn/stats"
	"log"
	"time"
)

func main() {
	processSerial()
}

func processSerial() {
	serial.Addr = "192.168.88.82:1234"
	serial.CheckConnection(true)

	mqttOpts := mqtt.Options{
		Address:  "192.168.88.209:1883",
		Username: "",
		Password: "",
		ClientId: "ninebot-mqtt",
		Topic:    "wifi2mqtt/%s",
	}

	mq, err := mqtt.Connect(mqttOpts)
	if err != nil {
		log.Fatalf("failed to connect to mqtt server: %s", err.Error())
	}
	var fullInfo = make(map[string]interface{}, 0)

	var capacityStats = make([]int16, 0)
	var capacityTimestamps = make([]time.Time, 0)
	var lastCapacity int16 = -1
	var capacityDrainRate float64
	var ttl int64 = 0
	var ttld time.Duration

	var serialNumber string

	for {
		statusReq := protocol.GetStatus()
		statusResp, err := scooter.Request(statusReq)
		if err != nil {
			log.Fatalf("status request error: %s", err.Error())
		}
		fullInfo["status"] = protocol.ToInt16(statusResp.Payload)

		serialNumberReq := protocol.GetSerialNumber()
		serialNumberResp, err := scooter.Request(serialNumberReq)
		if err != nil {
			log.Fatalf("serialNumber request error: %s", err.Error())
		}
		serialNumber = string(serialNumberResp.Payload)

		remainingCapacityPercReq := protocol.GetRemainingCapacityPerc()
		remainingCapacityPercResp, err := scooter.Request(remainingCapacityPercReq)
		if err != nil {
			log.Fatalf("remainingCapacityPerc request error: %s", err.Error())
		}
		fullInfo["remaining_capacity_perc"] = protocol.ToInt16(remainingCapacityPercResp.Payload)

		remainingCapacityReq := protocol.GetRemainingCapacity()
		remainingCapacityResp, err := scooter.Request(remainingCapacityReq)
		if err != nil {
			log.Fatalf("remainingCapacity request error: %s", err.Error())
		}
		fullInfo["remaining_capacity"] = protocol.ToInt16(remainingCapacityResp.Payload)

		if _, ok := fullInfo["actual_capacity"]; !ok {
			actualCapacityReq := protocol.GetActualCapacity()
			actualCapacityResp, err := scooter.Request(actualCapacityReq)
			if err != nil {
				log.Fatalf("actualCapacity request error: %s", err.Error())
			}
			fullInfo["actual_capacity"] = protocol.ToInt16(actualCapacityResp.Payload)

			factoryCapacityReq := protocol.GetFactoryCapacity()
			factoryCapacityResp, err := scooter.Request(factoryCapacityReq)
			if err != nil {
				log.Fatalf("factoryCapacity request error: %s", err.Error())
			}
			fullInfo["factory_capacity"] = protocol.ToInt16(factoryCapacityResp.Payload)
			if fullInfo["actual_capacity"] != fullInfo["factory_capacity"] {
				log.Printf("WARNING! factory capacity: %d, actual: %d", fullInfo["factory_capacity"], fullInfo["actual_capacity"])
				fullInfo["actual_capacity"] = fullInfo["factory_capacity"]
			}
		}

		currentReq := protocol.GetCurrent()
		currentResp, err := scooter.Request(currentReq)
		if err != nil {
			log.Fatalf("current request error: %s", err.Error())
		}
		fullInfo["current"] = float64(protocol.ToInt16(currentResp.Payload)) * 10 / 1000

		voltageReq := protocol.GetVoltage()
		voltageResp, err := scooter.Request(voltageReq)
		if err != nil {
			log.Fatalf("voltage request error: %s", err.Error())
		}
		fullInfo["voltage"] = float64(protocol.ToInt16(voltageResp.Payload)) * 10 / 1000
		fullInfo["power"] = fullInfo["current"].(float64) * fullInfo["voltage"].(float64)

		temperatureReq := protocol.GetTemperature()
		temperatureResp, err := scooter.Request(temperatureReq)
		if err != nil {
			log.Fatalf("temperature request error: %s", err.Error())
		}
		fullInfo["temperature_0"] = int(temperatureResp.Payload[0]) - 20
		fullInfo["temperature_1"] = int(temperatureResp.Payload[1]) - 20

		if lastCapacity == -1 {
			lastCapacity = fullInfo["remaining_capacity"].(int16)
		} else {
			capacityStats = append(capacityStats, lastCapacity-fullInfo["remaining_capacity"].(int16))
			capacityTimestamps = append(capacityTimestamps, time.Now())
			lastCapacity = fullInfo["remaining_capacity"].(int16)

			sum, _ := stats.LoadRawData(capacityStats).Sum()
			capacityDrainRate = sum / float64(time.Now().Unix()-capacityTimestamps[0].Unix())

			ttl = int64(float64(fullInfo["remaining_capacity"].(int16)) / capacityDrainRate)
			ttld, _ = time.ParseDuration(fmt.Sprintf("%ds", ttl))
			fullInfo["ttl"] = ttld.Seconds()
		}
		fullInfo["moving_avg_size"] = len(capacityStats)
		if len(capacityStats) > 10000 {
			capacityStats = append(capacityStats[:1], capacityStats[2:]...)
			capacityTimestamps = append(capacityTimestamps[:1], capacityTimestamps[2:]...)
		}

		fmt.Printf(
			"%s [%s] [%d%%] [%d/%dmAh] status: %#b; %05.2fA, %05.2fV, %05.2fW; %d°/%d°; ttl %05.2fh (%d)\r",
			time.Now().Format("2006-01-02 15:04:05"),
			serialNumber,
			fullInfo["remaining_capacity_perc"],
			fullInfo["remaining_capacity"],
			fullInfo["actual_capacity"],
			fullInfo["status"],
			fullInfo["current"],
			fullInfo["voltage"],
			fullInfo["power"],
			fullInfo["temperature_0"].(int),
			fullInfo["temperature_1"].(int),
			ttld.Hours(),
			fullInfo["moving_avg_size"],
		)
		mq.SendFullInfo(serialNumber, fullInfo)
	}

}
