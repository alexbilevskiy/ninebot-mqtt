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

	var serialNumber = ""

	for {
		statusReq := protocol.GetStatus()
		statusResp := scooter.Request(statusReq)
		fullInfo["status"] = protocol.ToInt16(statusResp.Payload)

		if serialNumber == "" {
			serialNumberReq := protocol.GetSerialNumber()
			serialNumberResp := scooter.Request(serialNumberReq)
			serialNumber = string(serialNumberResp.Payload)
		}

		remainingCapacityPercReq := protocol.GetRemainingCapacityPerc()
		remainingCapacityPercResp := scooter.Request(remainingCapacityPercReq)
		fullInfo["remaining_capacity_perc"] = protocol.ToInt16(remainingCapacityPercResp.Payload)

		remainingCapacityReq := protocol.GetRemainingCapacity()
		remainingCapacityResp := scooter.Request(remainingCapacityReq)
		fullInfo["remaining_capacity"] = protocol.ToInt16(remainingCapacityResp.Payload)

		if _, ok := fullInfo["actual_capacity"]; !ok {
			actualCapacityReq := protocol.GetActualCapacity()
			actualCapacityResp := scooter.Request(actualCapacityReq)
			fullInfo["actual_capacity"] = protocol.ToInt16(actualCapacityResp.Payload)

			factoryCapacityReq := protocol.GetFactoryCapacity()
			factoryCapacityResp := scooter.Request(factoryCapacityReq)
			fullInfo["factory_capacity"] = protocol.ToInt16(factoryCapacityResp.Payload)
			if fullInfo["actual_capacity"] != fullInfo["factory_capacity"] {
				log.Printf("WARNING! factory capacity: %d, actual: %d", fullInfo["factory_capacity"], fullInfo["actual_capacity"])
				fullInfo["actual_capacity"] = fullInfo["factory_capacity"]
			}
		}

		currentReq := protocol.GetCurrent()
		currentResp := scooter.Request(currentReq)
		fullInfo["current"] = float64(protocol.ToInt16(currentResp.Payload)) * 10 / 1000

		voltageReq := protocol.GetVoltage()
		voltageResp := scooter.Request(voltageReq)
		fullInfo["voltage"] = float64(protocol.ToInt16(voltageResp.Payload)) * 10 / 1000
		fullInfo["power"] = fullInfo["current"].(float64) * fullInfo["voltage"].(float64)

		if len(capacityStats)%10 == 0 {
			cellsVoltageReq := protocol.GetCellsVoltage()
			cellsVoltageResp := scooter.Request(cellsVoltageReq)
			fullInfo["cell_voltage"] = scooter.ParseCellsVoltageResp(cellsVoltageResp.Payload)
		}

		temperatureReq := protocol.GetTemperature()
		temperatureResp := scooter.Request(temperatureReq)
		fullInfo["temperature"] = make(map[string]int, 2)
		fullInfo["temperature"].(map[string]int)["zone_0"] = int(temperatureResp.Payload[0]) - 20
		fullInfo["temperature"].(map[string]int)["zone_1"] = int(temperatureResp.Payload[1]) - 20

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
			fullInfo["temperature"].(map[string]int)["zone_0"],
			fullInfo["temperature"].(map[string]int)["zone_1"],
			ttld.Hours(),
			fullInfo["moving_avg_size"],
		)
		mq.SendFullInfo(serialNumber, fullInfo)
	}

}
