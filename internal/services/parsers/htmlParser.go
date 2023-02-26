package parsers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"goParser/internal/config"
	"goParser/internal/services"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	HOST                     = "https://serbiarus.com"
	STORAGE_TABLE            = "storage2"
	TAG_NAME_PATTERN         = "regexp"
	TAG_NAME_PATTERN_WRAPPER = "wrapper"
)

type OfferDataStruct struct {
	Title       string   `regexp:"1:(?is)<title>(.+)</title>"`
	Type        string   `regexp:"1:(?is)<div>Тип предложения: <b>(.+?)</b></div>"`
	Territory   string   `regexp:"1:(?is)<div>Территориально: <b>(.+?)</b></div>"`
	PriceFrom   string   `regexp:"1:(?is)<div>Стоимость: от <b>(.+?)</b></div>"`
	Phone       string   `regexp:"1:(?is)Телефон: <a href='tel:(.+?)'>"`
	Telegram    string   `regexp:"1:(?is)Телеграм: <noindex><a href='.*' target='_blank'>(.+?)</a></noindex>"`
	WhatsApp    string   `regexp:"1:(?is)WhatsApp: <noindex><a href='.*' target='_blank'>(.+?)</a></noindex>"`
	Instagram   string   `regexp:"1:(?is)Instagram: <noindex><a href='.*' target='_blank'>(.+?)</a></noindex>"`
	Viber       string   `regexp:"1:(?is)Viber: <noindex><a href='.*'>(.+?)</a></noindex>"`
	Facebook    string   `regexp:"1:(?is)Facebook: <noindex><a href='.*' target='_blank'>(.+?)</a></noindex>"`
	Categories  []string `regexp:"2:(?is)(<a .*?>(.+?)</a>)" wrapper:"1:(?is)<div>Категории: <b>(.*?)</b></div>"`
	Languages   []string `regexp:"1:(?is)(.+)(, )*" wrapper:"1:(?is)<div>Языки: <b>(.*?)</b></div>"`
	OfferText   string   `regexp:"1:(?is)(.+)" wrapper:"1:(?is)<div style=\"margin-bottom:10px; font-size:16px; word-break: break-all; word-break: break-word;\">(.*?)</div>"`
	AuthorName  string   `regexp:"1:(?is)(.+)" wrapper:"1:(?is)<div .*?>.*?<a href=\".*?\" target=\"_blank\"><b>(.*?)</b></a>.*?<div style=\"font-size:12px;\">"`
	ImagesSmall []string `regexp:"1:(?is)<img data-src=\"(.+?)\" class=\"offer_img\"" wrapper:"1:(?is)<div class=\"d_narrow_page\">(.*?)<img class=\"user_small_avatar\""`
	Avatar      string   `regexp:"1:(?is)<img class=\"user_small_avatar\" data-src=\"(.+?)\">"`
}

func HtmlParser(streamLimit int, client *http.Client, forceMode bool, dataStruct interface{}) {
	fmt.Println("******Hello! I'm parser vmeste******")
	db, _ := services.InitDb(config.DbConfig(), config.DbSchema())
	defer db.Close()

	var rowInsertedCount int64 = 0
	var rowCount int64 = 0
	var offersUrls []string

	// Бежим по страницам
	for page := 1; true; page++ {
		// Запрос посылаем на сервер в однопотоке, чтобы не перегружать сеть и сервер
		fmt.Printf("Start processing page %d\n", page)
		var targetUrl string = fmt.Sprintf("https://serbiarus.com/country/serbia/offers/page-%d.html", page)

		// Форимируем запрос
		request, err := http.NewRequest("GET", targetUrl, nil)
		if err != nil {
			log.Println(err)
		}

		// Запрашиваем
		response, err := client.Do(request)
		if err != nil {
			log.Println(err)
		}

		// Читаем
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println(err)
		}

		content := string(data)
		re, _ := regexp.Compile(`(?is)class="offer_list_block".*?<a href="(/country.+?)"`)
		res := re.FindAllStringSubmatch(content, -1)

		var haveOffers bool = false
		for _, item := range res {
			haveOffers = true
			offersUrls = append(offersUrls, HOST+item[1])
		}

		if !haveOffers {
			break
		}

		//break // пока хватит одной страницы
	}

	dataForStreams := splitData(offersUrls, streamLimit)

	// Бежим по предложениям асинхронно
	var wg sync.WaitGroup
	for streamId := 0; streamId < streamLimit; streamId++ {
		wg.Add(1)
		go func(streamId int) {
			defer wg.Done()
			for _, currentOfferUrl := range dataForStreams[streamId] {
				rowCount++
				offerHash := md5.New()
				offerHashId := hex.EncodeToString(offerHash.Sum([]byte(currentOfferUrl)))
				if !forceMode && services.CheckExistsById(db, STORAGE_TABLE, offerHashId) {
					fmt.Printf("Stream %d: Skip offer %s. Already parsed!\n", streamId, currentOfferUrl)
					continue
				}
				fmt.Printf("Stream %d: Start processing offer %s\n", streamId, currentOfferUrl)

				// Форимируем запрос
				request, err := http.NewRequest("GET", currentOfferUrl, nil)
				if err != nil {
					log.Println(err)
				}

				// Запрашиваем
				response, err := client.Do(request)
				if err != nil {
					log.Println(err)
				}

				// Читаем
				data, err := ioutil.ReadAll(response.Body)
				if err != nil {
					log.Println(err)
				}

				content := string(data)
				offerData := dataStruct
				typeReflect := reflect.TypeOf(offerData)

				dataRecord := reflect.ValueOf(&offerData).Elem()
				valueReflect := reflect.New(dataRecord.Elem().Type()).Elem()
				valueReflect.Set(dataRecord.Elem())
				for i := 0; i < typeReflect.NumField(); i++ {
					field := typeReflect.Field(i)

					var wrappedContent string
					wrapper, ok := field.Tag.Lookup(TAG_NAME_PATTERN_WRAPPER)
					if ok {
						wrapperArr := strings.SplitN(wrapper, ":", 2)
						wrapperIndex, _ := strconv.Atoi(wrapperArr[0])
						wrapperPattern := wrapperArr[1]

						re, _ := regexp.Compile(wrapperPattern)
						res := re.FindAllStringSubmatch(content, -1)
						if len(res) > 0 && len(res[0]) > wrapperIndex {
							wrappedContent = res[0][wrapperIndex]
						} else {
							wrappedContent = ""
						}
					} else {
						wrappedContent = content
					}

					patternArr := strings.SplitN(field.Tag.Get(TAG_NAME_PATTERN), ":", 2)
					index, _ := strconv.Atoi(patternArr[0])
					pattern := patternArr[1]

					re, _ := regexp.Compile(pattern)
					res := re.FindAllStringSubmatch(wrappedContent, -1)
					if len(res) > 0 && len(res[0]) > 1 {
						switch field.Type {
						case reflect.TypeOf(""):
							valueReflect.Field(i).SetString(res[0][index])
						case reflect.TypeOf([]string{}):
							var tempArr []string
							for _, item := range res {
								tempArr = append(tempArr, item[index])
							}
							valueReflect.Field(i).Set(reflect.ValueOf(tempArr))
						}
						dataRecord.Set(valueReflect)
					}
				}

				//fmt.Println(offerData)

				// Сохраняем
				jsonStr, _ := json.Marshal(offerData)
				currentTime := time.Now()
				insetResult, _ := services.InsertToStorage(db, STORAGE_TABLE, forceMode, []services.DbStorageStruct{
					{
						Id:                   offerHashId,
						LastUpdateInDb:       "",
						LastUpdateFromSource: currentTime.Format("2006-01-02 15:04:05"),
						Data:                 string(jsonStr),
					},
				})

				if insetResult != nil {
					inserted, _ := insetResult.RowsAffected()
					rowInsertedCount += inserted
				}

				//break // пока хватит одной страницы
			}
		}(streamId)
	}

	fmt.Printf("Waiting... ")
	wg.Wait()
	fmt.Printf("Stored %d records from %d\n", rowInsertedCount, rowCount)
}

func splitData(data []string, count int) [][]string {
	preparedData := make([][]string, count)

	for index, currentOfferUrl := range data {
		i := index % count
		preparedData[i] = append(preparedData[i], currentOfferUrl)
	}

	return preparedData
}
