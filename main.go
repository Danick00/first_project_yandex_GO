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
	pollInterval = 5 * time.Second // Интервал опроса
)

func main() {
	errorCount := 0
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	fmt.Println("Скрипт мониторинга запущен...")

	for range ticker.C {
		data, err := fetchStats(client)
		if err != nil {
			errorCount++
			if errorCount >= 3 {
				fmt.Println("Unable to fetch server statistic")
			}
			continue
		}

		// Если данные получены успешно, сбрасываем счетчик ошибок
		errorCount = 0
		checkThresholds(data)
	}
}

// fetchStats делает запрос и возвращает слайс чисел или ошибку
func fetchStats(client *http.Client) ([]float64, error) {
	resp, err := client.Get(statsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Убираем лишние пробелы и разбиваем строку по запятой
	lines := strings.Split(strings.TrimSpace(string(body)), ",")
	if len(lines) != 7 {
		return nil, fmt.Errorf("invalid data format")
	}

	// Конвертируем строки в числа float64
	var values []float64
	for _, s := range lines {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}

	return values, nil
}

// checkThresholds проверяет значения и выводит алерты
func checkThresholds(v []float64) {
	// Индексы согласно заданию:
	// 0: Load Avg, 1: RAM Total, 2: RAM Used, 3: Disk Total, 4: Disk Used, 5: Net Cap, 6: Net Load

	// 1. Load Average
	if v[0] > 30 {
		fmt.Printf("Load Average is too high: %v\n", v[0])
	}

	// 2. RAM Usage (80%)
	ramUsagePercent := (v[2] / v[1]) * 100
	if ramUsagePercent > 80 {
		fmt.Printf("Memory usage too high: %.0f%%\n", ramUsagePercent)
	}

	// 3. Disk Space (90%)
	diskUsagePercent := (v[4] / v[3]) * 100
	if diskUsagePercent > 90 {
		freeDiskMb := (v[3] - v[4]) / (1024 * 1024)
		fmt.Printf("Free disk space is too low: %.0f Mb left\n", freeDiskMb)
	}

	// 4. Network Bandwidth (90%)
	netUsagePercent := (v[6] / v[5]) * 100
	if netUsagePercent > 90 {
		// Свободная полоса в Мбит/с: (Байты в сек * 8) / 1 000 000
		freeNetMbit := ((v[5] - v[6]) * 8) / 1000000
		fmt.Printf("Network bandwidth usage high: %.0f Mbit/s available\n", freeNetMbit)
	}
}
