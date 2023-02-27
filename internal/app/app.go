package app

import (
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"goParser/internal/services"
	"goParser/internal/services/parsers"
	"golang.org/x/net/proxy"
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
		proxyAddr, exists := os.LookupEnv("PROXY_ADDR")
		if exists {
			fmt.Printf("Use proxy %s is active!\n", proxyAddr)
			proxyAuthString, existsAuth := os.LookupEnv("PROXY_AUTH")
			if existsAuth {
				authData := strings.Split(proxyAuthString, ":")
				proxyAuth := proxy.Auth{User: authData[0], Password: authData[1]}
				client, _ = services.ProxyCon(proxyAddr, &proxyAuth, 30)
			} else {
				client, _ = services.ProxyCon(proxyAddr, nil, 30)
			}
		}
	}

	var parsersListArray = strings.Split(*parsersList, ",")

	for _, parserName := range parsersListArray {
		switch strings.Trim(parserName, " ") {
		case "serbiarus":
			parsers.HtmlParser(*streamCount, client, *useForceFlag, parsers.OfferDataStruct{}, parsers.HtmlParserConfigStruct{
				"Serbiarus",
				"https://serbiarus.com/country/serbia/offers/page-%d.html",
				1,
				`(?is)class="offer_list_block".*?<a href="(/country.+?)"`,
			})
		case "vmeste":
			parsers.JsonApiParser(100, client, *useForceFlag)
		default:
			fmt.Printf("Parser with name '%s' not found!\n", strings.Trim(parserName, " "))
		}
	}
}
