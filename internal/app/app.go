package app

import (
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"goParser/internal/services"
	"goParser/internal/services/parsers"
	"net/http"
	"os"
	"strings"
)

func Run() {
	client := &http.Client{}

	// Обработка аргументов командной строки
	useProxyFlag := flag.Bool("proxy", false, "Need use proxy?")
	useForceFlag := flag.Bool("force", false, "Do not skip the already processed data?")
	streamCount := flag.Int("stream-count", 1, "The number of streams for parsing")
	parsersList := flag.String("parsers", "serbiarus, vmeste", "List parsers for invoke")
	flag.Parse()

	if *useProxyFlag {
		proxyString, exists := os.LookupEnv("PROXY_STRING")
		if exists {
			fmt.Printf("Use proxy %s is active!", proxyString)
			client, _ = services.ProxyCon(proxyString, 30)
		}
	}

	var parsersListArray = strings.Split(*parsersList, ",")

	for _, parserName := range parsersListArray {
		switch strings.Trim(parserName, " ") {
		case "serbiarus":
			parsers.HtmlParser(*streamCount, client, *useForceFlag, parsers.OfferDataStruct{})
		case "vmeste":
			parsers.JsonApiParser(100, client, *useForceFlag)
		default:
			fmt.Printf("Parser with name '%s' not found!\n", strings.Trim(parserName, " "))
		}
	}
}
