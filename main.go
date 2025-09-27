package main

import (
	"fmt"
	"io"		// используем для чтения тела HTTP-ответа (io.ReadAll)
	//"log"
	"net/http"	// для выполнения HTTP-запросов
	"strconv"
	"strings"
	"time"
)


// объявление параметров и порогов срабатывания
const (
	targetURL       = "http://srv.msk01.gigacorp.local/_stats"
	checkInterval   = 5 * time.Second	// интервал опроса
	maxFailures     = 3					// максимальное количество ошибок подряд
	loadAvgThreshold = 30.0
	memUsageThreshold = 0.8				// 80%
	diskUsageThreshold = 0.9			// 90%
	netUsageThreshold = 0.9				// 90%
)

func main() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	failCount := 0

	for {
		resp, err := client.Get(targetURL)
		if err != nil { // если не удалось подключиться, увеличиваем счётчик ошибок
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
				failCount = 0 // сбросить счётчик после вывода сообщения в терминал
			}
			time.Sleep(checkInterval)
			continue // откатываемся в начало цикла
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {  // если статус не 200, то увеличиваем счётчик ошибок
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
				failCount = 0
			}
			time.Sleep(checkInterval)
			continue // откатываемся в начало цикла
		}

		if err != nil {
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
				failCount = 0
			}
			time.Sleep(checkInterval)
			continue // откатываемся в начало цикла
		}

		fields := strings.Split(strings.TrimSpace(string(body)), ",")
		if len(fields) != 7 {
			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
				failCount = 0
			}
			time.Sleep(checkInterval)
			continue
		}

		// Сброс счётчика ошибок при успешном получении данных
		failCount = 0

		// Парсинг значений
		loadAvg, err1 := strconv.ParseFloat(fields[0], 64)
		totalMem, err2 := strconv.ParseInt(fields[1], 10, 64)
		usedMem, err3 := strconv.ParseInt(fields[2], 10, 64)
		totalDisk, err4 := strconv.ParseInt(fields[3], 10, 64)
		usedDisk, err5 := strconv.ParseInt(fields[4], 10, 64)
		totalNet, err6 := strconv.ParseInt(fields[5], 10, 64)
		usedNet, err7 := strconv.ParseInt(fields[6], 10, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil ||
			err5 != nil || err6 != nil || err7 != nil {
			// ошибка парсинга — считаем как ошибку получения данных

			failCount++
			if failCount >= maxFailures {
				fmt.Println("Unable to fetch server statistic")
				failCount = 0
			}
			time.Sleep(checkInterval)
			continue
		}

		// Проверка Load Average
		if loadAvg > loadAvgThreshold {
			fmt.Printf("Load Average is too high: %.0f\n", loadAvg)
		}

		// Проверка памяти (> 80% = 4/5)
		if totalMem > 0 && usedMem*5 > totalMem*4 {
			percentage := int((float64(usedMem) / float64(totalMem)) * 100)
			fmt.Printf("Memory usage too high: %d%%\n", percentage)
		}

		// Проверка диска (> 90% = 9/10)
		if totalDisk > 0 && usedDisk*10 > totalDisk*9 {
			freeBytes := totalDisk - usedDisk
			freeMB := freeBytes / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
		}

		if totalNet > 0 {
			netUsage := float64(usedNet) / float64(totalNet)
			if netUsage > netUsageThreshold {
				freeBytesPerSec := totalNet - usedNet
				// Переводим байты/сек в мегабиты/сек: *8 / 1000 / 1000
				freeMbitPerSec := float64(freeBytesPerSec) * 8 / 1_000_000
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", int(freeMbitPerSec))
			}
		}

		time.Sleep(checkInterval)
		
	}
}