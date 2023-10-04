package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	latencyHistory = 1800 // 1 hour with 2-second intervals
)

var (
	latencies []map[string]interface{}
	labels    []string
	upgrader  = websocket.Upgrader{}
)

func createChart() string {
	data := map[string]interface{}{
		"title": map[string]string{
			"text": "Latency over time",
		},
		"xAxis": []map[string]interface{}{
			{
				"data": labels,
			},
		},
		"series": []map[string]interface{}{
			{
				"data": latencies,
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		time.Sleep(2 * time.Second)
		output, err := exec.Command("ping", "-n", "1", "1.1.1.1").Output()

		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Ping error: "+err.Error()))

			// Add a 0ms latency data with red color to signify error.
			latencies = append(latencies, map[string]interface{}{
				"value":     0,
				"itemStyle": map[string]string{"color": "red"},
			})
			labels = append(labels, time.Now().Format("15:04:05"))

		} else {
			re := regexp.MustCompile(`Average = (\d+)ms`)
			matches := re.FindStringSubmatch(string(output))
			if len(matches) == 2 {
				avgLatency, _ := strconv.ParseFloat(matches[1], 64)

				if len(latencies) >= latencyHistory {
					latencies = latencies[1:]
					labels = labels[1:]
				}

				latencies = append(latencies, map[string]interface{}{
					"value": avgLatency,
				})
				labels = append(labels, time.Now().Format("15:04:05"))
			}
		}

		err = conn.WriteMessage(websocket.TextMessage, []byte(createChart()))
		if err != nil {
			return
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWebSocketConnection)
	http.Handle("/", http.FileServer(http.Dir("static")))

	fmt.Println("Server started at http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}
