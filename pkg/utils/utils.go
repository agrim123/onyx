package utils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/agrim123/onyx/pkg/logger"
)

func GetPublicIP() string {
	url := "https://api.ipify.org?format=text" // we are using a pulib IP API, we're using ipify here, below are some others
	// https://www.ipify.org
	// http://myexternalip.com/raw
	// http://ident.me
	// http://whatismyipaddress.com/api
	logger.Info("Getting IP address from ipify")
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s/32", string(ip))
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
