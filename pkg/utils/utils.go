package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"

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
