package utils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"bitbucket.org/agrim123/onyx/pkg/logger"
)

var sources = []string{
	"https://api.ipify.org?format=text",
	"https://api64.ipify.org/?format=text",
	"https://www.ipify.org",
	"https://myexternalip.com/raw",
}

func getIP() string {
	for _, source := range sources {
		logger.Info("Getting IP address from %s", logger.Underline(source))
		resp, err := http.Get(source)
		if err != nil {
			logger.Error("Unable to get ip from %s. Error: %v", logger.Underline(source), err)
			continue
		}

		defer resp.Body.Close()
		ip, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Unable to read ip from response %s. Error: %s", logger.Underline(source), err.Error())
			continue
		} else {
			return string(ip)
		}
	}

	return ""
}

func GetPublicIP() string {
	ip := getIP()
	if ip == "" {
		logger.Fatal("Unable to determine ip")
	}

	cidr := fmt.Sprintf("%s/32", ip)

	logger.Success("Authorizing for CIDR: " + cidr)

	return cidr
}

func GetChunks(arr []string, chunkSize int) [][]string {
	if len(arr) == 0 {
		return nil
	}
	chunks := make([][]string, (len(arr)+chunkSize-1)/chunkSize)
	prev := 0
	i := 0

	for prev < len(arr)-chunkSize {
		next := prev + chunkSize
		chunks[i] = arr[prev:next]
		prev = next
		i++
	}

	chunks[i] = arr[prev:]
	return chunks
}

func GetUserInput(message string) string {
	consoleReader := bufio.NewReader(os.Stdin)
	fmt.Print(message)
	input, _ := consoleReader.ReadString('\n')
	return input
}
