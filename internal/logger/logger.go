package logger

import (
	"encoding/json"
	"fmt"
	"job-queue/internal/task"
	"log"
	"os"
)

func LogSuccess(cur_task task.Task) {
	f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatal("Error logging: ", err)
		return
	}
	defer f.Close()

	payload_str, er := json.Marshal(cur_task.Payload)
	if er != nil {
		payload_str = []byte{}
	}

	text := "\n SUCCESS: Task type: " + cur_task.Type + " Task Payload: " + string(payload_str) + " Retries left: " + fmt.Sprint(cur_task.Retries) + "\n"

	if _, err := f.WriteString(text); err != nil {
		log.Fatal("Error writing to the log file: ", err)
		return
	}
	log.Println("logged successfully to the file")
}

func LogFailure(cur_task task.Task, curr_err error) {
	f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatal("Error logging: ", err)
		return
	}
	defer f.Close()

	payload_str, er := json.Marshal(cur_task.Payload)
	if er != nil {
		payload_str = []byte{}
	}

	text := "FAILURE: Task type: " + cur_task.Type + " Task payload: " + string(payload_str) + " Retries left: " + fmt.Sprintf("%d", cur_task.Retries) + " Error message: " + curr_err.Error() + "\n"

	if _, err := f.WriteString(text); err != nil {
		log.Fatal("Error writing to the log file: ", err)
		return
	}
	log.Println("logged successfully to the file")

}
