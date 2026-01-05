package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	statsURL     = "http://srv.msk01.gigacorp.local/_stats"
	pollInterval = 5 * time.Second
)

func main() {
	errorCount := 0
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		data, err := fetchStats(client)
		if err != nil {
			errorCount++
			if errorCount >= 3 {
				fmt.Println("Unable to fetch server statistic")
			}
		} else {
			errorCount = 0
			checkThresholds(data)
		}
		time.Sleep(pollInterval)
	}
}

func fetchStats(client *http.Client) ([]float64, error) {
	resp, err := client.Get(statsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(string(body)), ",")
	var values []float64
	for _, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}

	if len(values) != 7 {
		return nil, fmt.Errorf("invalid data")
	}

	return values, nil
}

func checkThresholds(v []float64) {
	// Индексы: 0:LA, 1:RAM_Tot, 2:RAM_Usd, 3:Disk_Tot, 4:Disk_Usd, 5:Net_Tot, 6:Net_Usd

	// 1. Load Average
	if v[0] > 30 {
		fmt.Printf("Load Average is too high: %v\n", v[0])
	}

	// 2. RAM Usage (> 80%)
	ramUsage := (v[2] * 100) / v[1]
	if ramUsage > 80 {
		fmt.Printf("Memory usage too high: %d%%\n", int(ramUsage))
	}

	// 3. Disk Space (> 90%)
	diskUsage := (v[4] * 100) / v[3]
	if diskUsage > 90 {
		freeMb := int((v[3] - v[4]) / (1024 * 1024))
		fmt.Printf("Free disk space is too low: %d Mb left\n", freeMb)
	}

	// 4. Network Bandwidth (> 90%)
	netUsage := (v[6] * 100) / v[5]
	if netUsage > 90 {
		// Расчет свободной полосы: (Емкость - Нагрузка) / 1 000 000
		freeMbit := int((v[5] - v[6]) / 1000000)
		fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", freeMbit)
	}
}
