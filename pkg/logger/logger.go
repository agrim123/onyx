package logger

import "fmt"

func Info(message string) {
	fmt.Println("[onyx] " + message)
}
