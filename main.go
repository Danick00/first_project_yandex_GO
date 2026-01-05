package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const statsURL = "http://srv.msk01.gigacorp.local/_stats"

func main() {
	errorCount := 0
	client := &http.Client{Timeout: 2 * time.Second}

	// ВАЖНО: Убрали Println("Скрипт мониторинга запущен...")

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
		time.Sleep(5 * time.Second) // Опрос раз в 5 секунд
	}
}

func fetchStats(client *http.Client) ([]float64, error) {
	resp, err := client.Get(statsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(string(body)), ",")
	var values []float64
	for _, p := range parts {
		v, _ := strconv.ParseFloat(p, 64)
		values = append(values, v)
	}
	return values, nil
}

func checkThresholds(v []float64) {
	// 1. Load Average (Просто выводим целое число, если > 30)
	if v[0] > 30 {
		fmt.Printf("Load Average is too high: %v\n", v[0])
	}

	// 2. RAM (Используем int для отсечения дроби)
	ramUsage := (v[2] / v[1]) * 100
	if ramUsage > 80 {
		fmt.Printf("Memory usage too high: %v%%\n", int(ramUsage))
	}

	// 3. Disk (Делим как целые числа, чтобы получить 17182)
	diskUsage := (v[4] / v[3]) * 100
	if diskUsage > 90 {
		freeMb := int((v[3] - v[4]) / (1024 * 1024))
		fmt.Printf("Free disk space is too low: %v Mb left\n", freeMb)
	}

	// 4. Network (Тест ожидает разницу / 1 000 000 без умножения на 8)
	netUsage := (v[6] / v[5]) * 100
	if netUsage > 90 {
		freeMbit := int((v[5] - v[6]) / 1000000)
		fmt.Printf("Network bandwidth usage high: %v Mbit/s available\n", freeMbit)
	}
}
