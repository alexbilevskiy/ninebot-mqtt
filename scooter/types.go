package scooter

type FullInfo struct {
	Status                int16            `json:"status"`
	RemainingCapacityPerc int16            `json:"remaining_capacity_perc"`
	RemainingCapacity     int16            `json:"remaining_capacity"`
	ActualCapacity        int16            `json:"actual_capacity"`
	FactoryCapacity       int16            `json:"factory_capacity"`
	Current               float64          `json:"current"`
	Voltage               float64          `json:"voltage"`
	Power                 float64          `json:"power"`
	CellVoltage           map[string]int16 `json:"cell_voltage"`
	Temperature           map[string]int   `json:"temperature"`
	Ttl                   float64          `json:"ttl"`
	MovingAvgSize         int              `json:"moving_avg_size"`
}
