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

	var fullInfo scooter.FullInfo

	var capacityStats = make([]int16, 0)
	var capacityTimestamps = make([]time.Time, 0)
	var lastCapacity int16 = -1
	var capacityDrainRate float64
	var ttl int64 = 0
	var ttld time.Duration
	var lastTs string
	var now time.Time

	var serialNumber = ""

	for {
		now = time.Now()
		statusReq := protocol.GetStatus()
		statusResp := scooter.Request(statusReq)
		fullInfo.Status = protocol.ToInt16(statusResp.Payload)

		if serialNumber == "" {
			serialNumberReq := protocol.GetSerialNumber()
			serialNumberResp := scooter.Request(serialNumberReq)
			serialNumber = string(serialNumberResp.Payload)
		}

		remainingCapacityPercReq := protocol.GetRemainingCapacityPerc()
		remainingCapacityPercResp := scooter.Request(remainingCapacityPercReq)
		fullInfo.RemainingCapacityPerc = protocol.ToInt16(remainingCapacityPercResp.Payload)

		remainingCapacityReq := protocol.GetRemainingCapacity()
		remainingCapacityResp := scooter.Request(remainingCapacityReq)
		fullInfo.RemainingCapacity = protocol.ToInt16(remainingCapacityResp.Payload)

		if fullInfo.ActualCapacity == 0 {
			actualCapacityReq := protocol.GetActualCapacity()
			actualCapacityResp := scooter.Request(actualCapacityReq)
			fullInfo.ActualCapacity = protocol.ToInt16(actualCapacityResp.Payload)

			factoryCapacityReq := protocol.GetFactoryCapacity()
			factoryCapacityResp := scooter.Request(factoryCapacityReq)
			fullInfo.FactoryCapacity = protocol.ToInt16(factoryCapacityResp.Payload)
			if fullInfo.ActualCapacity != fullInfo.FactoryCapacity {
				log.Printf("WARNING! factory capacity: %d, actual: %d", fullInfo.FactoryCapacity, fullInfo.ActualCapacity)
				fullInfo.ActualCapacity = fullInfo.FactoryCapacity
			}
		}

		currentReq := protocol.GetCurrent()
		currentResp := scooter.Request(currentReq)
		fullInfo.Current = float64(protocol.ToInt16(currentResp.Payload)) * 10 / 1000

		voltageReq := protocol.GetVoltage()
		voltageResp := scooter.Request(voltageReq)
		fullInfo.Voltage = float64(protocol.ToInt16(voltageResp.Payload)) * 10 / 1000
		fullInfo.Power = fullInfo.Current * fullInfo.Voltage

		if len(capacityStats)%10 == 0 {
			cellsVoltageReq := protocol.GetCellsVoltage()
			cellsVoltageResp := scooter.Request(cellsVoltageReq)
			fullInfo.CellVoltage = scooter.ParseCellsVoltageResp(cellsVoltageResp.Payload)
		}

		temperatureReq := protocol.GetTemperature()
		temperatureResp := scooter.Request(temperatureReq)
		fullInfo.Temperature = make(map[string]int, 2)
		fullInfo.Temperature["zone_0"] = int(temperatureResp.Payload[0]) - 20
		fullInfo.Temperature["zone_1"] = int(temperatureResp.Payload[1]) - 20

		if lastCapacity == -1 {
			lastCapacity = fullInfo.RemainingCapacity
		} else {
			diff := lastCapacity - fullInfo.RemainingCapacity
			if diff < 0 {
				diff = 0
			}
			capacityStats = append(capacityStats, diff)
			capacityTimestamps = append(capacityTimestamps, now)
			lastCapacity = fullInfo.RemainingCapacity

			sum, _ := stats.LoadRawData(capacityStats).Sum()
			capacityDrainRate = sum / float64(now.Unix()-capacityTimestamps[0].Unix())

			ttl = int64(float64(fullInfo.RemainingCapacity) / capacityDrainRate)
			ttld, _ = time.ParseDuration(fmt.Sprintf("%ds", ttl))
			fullInfo.Ttl = ttld.Seconds()
			lastTs = capacityTimestamps[0].Format("2006-01-02 15:04:05")
		}
		if len(capacityStats) > 10000 {
			capacityStats = append(capacityStats[:1], capacityStats[2:]...)
			capacityTimestamps = append(capacityTimestamps[:1], capacityTimestamps[2:]...)
		}
		fullInfo.MovingAvgSize = len(capacityStats)

		fmt.Printf(
			"%s (since %s) [%s] [%d%%] [%d/%dmAh] status: %#b; %05.2fA, %05.2fV, %05.2fW; %d°/%d°; ttl %05.2fh (%d)\r",
			now.Format("2006-01-02 15:04:05"),
			lastTs,
			serialNumber,
			fullInfo.RemainingCapacityPerc,
			fullInfo.RemainingCapacity,
			fullInfo.ActualCapacity,
			fullInfo.Status,
			fullInfo.Current,
			fullInfo.Voltage,
			fullInfo.Power,
			fullInfo.Temperature["zone_0"],
			fullInfo.Temperature["zone_1"],
			ttld.Hours(),
			fullInfo.MovingAvgSize,
		)
		mq.SendFullInfo(serialNumber, fullInfo)
	}

}
