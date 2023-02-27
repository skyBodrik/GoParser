package parsers

import (
	"encoding/json"
	"fmt"
	"goParser/internal/config"
	"goParser/internal/services"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type RootField struct {
	Result ResultField `json:"result"`
}

type ResultField struct {
	Data Data1Field `json:"data"`
}

type Data1Field struct {
	Json JsonField `json:"json"`
}

type JsonField struct {
	Data        []map[string]any `json:"data"`
	CountByType map[string]int   `json:"countByType"`
	NextCursor  int              `json:"nextCursor"`
	NextPage    int              `json:"nextPage"`
}

func JsonApiParser(parseLimit int, client *http.Client, forceMode bool) {
	fmt.Println("******Hello! I'm parser vmeste******")
	db, _ := services.InitDb(config.DbConfig(), config.DbSchema())
	defer db.Close()

	var cursor int = 0
	var rowInsertedCount int64 = 0
	var rowCount int64 = 0
	var duplicateCounter int = 0
	var lastUpdateTime string = services.GetLastUpdateTime(db, "storage1")
	insertedCounters := make(map[int]chan int, 1)
	totalCounters := make(map[int]chan int, 1)
	duplicateCounters := make(map[int]chan int, 1)

exit:
	for true {
		// Запрос посылаем на сервер в однопотоке, чтобы не перегружать сеть и сервер
		fmt.Printf("Start processing page %d\n", cursor)
		var targetUrl string = "https://vmeste.info/api/trpc/announcement.list?batch=1&input="

		targetUrl = targetUrl + url.QueryEscape(`{"0":{"json":{"filters":{"groupId":"5837698c-0957-42b3-b5cb-f6b07b81d3c1","payment":[],"isOnline":true,"isEditorsChoice":false,"location":null,"rubric":null,"type":"offers","searchSlug":"","orderBy":"date","isCommercial":false,"isNonCommercial":false,"locations":[],"rubrics":[],"inSearch":true,"suggesting":false},"limit":`+strconv.Itoa(parseLimit)+`,"orderBy":{"sort":"date","order":"DESC"},"cursor":`+strconv.Itoa(cursor)+`}}}`)
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
		//fmt.Println(data)

		// Отджейсоним полученные данные
		var result []RootField
		json.Unmarshal(data, &result)

		contentData := result[0].Result.Data.Json

		if !forceMode {
			for _, item := range contentData.Data {
				if strings.Compare(lastUpdateTime, item["updated"].(string)) >= 0 {
					// Если нечего обновлять, то не обновляем
					break exit
				}
				break
			}
		}

		// Обработку полученных данных осуществляем в потоках
		insertedCounters[cursor] = make(chan int, 1)
		totalCounters[cursor] = make(chan int, 1)
		duplicateCounters[cursor] = make(chan int, 1)
		go func(insertedCounter chan int, totalCounter chan int, duplicateCounter chan int) {
			insertedCount := 0
			totalCount := 0
			duplicateCount := 0
			for _, item := range contentData.Data {
				totalCount++
				//fmt.Println(key, item["id"], item["updated"])
				jsonStr, _ := json.Marshal(item)
				insetResult, err := services.InsertToStorage(db, "storage1", forceMode, []services.DbStorageStruct{
					{
						Id:                   item["id"].(string),
						LastUpdateInDb:       "",
						LastUpdateFromSource: item["updated"].(string),
						Data:                 string(jsonStr),
					},
				})

				if insetResult != nil {
					inserted, _ := insetResult.RowsAffected()
					insertedCount += int(inserted)
				}

				if err != nil {
					duplicateCount++
				}
			}

			insertedCounter <- insertedCount
			totalCounter <- totalCount
			duplicateCounter <- duplicateCount
		}(insertedCounters[cursor], totalCounters[cursor], duplicateCounters[cursor])

		if cursor == contentData.NextCursor || contentData.NextCursor == 0 {
			break
		}

		cursor = contentData.NextCursor
	}

	fmt.Printf("Waiting... ")
	for _, counter := range insertedCounters {
		rowInsertedCount += int64(<-counter)
	}
	for _, counter := range totalCounters {
		rowCount += int64(<-counter)
	}
	for _, counter := range duplicateCounters {
		duplicateCounter += <-counter
	}

	fmt.Printf("Stored %d records. DuplicatesOrErrors %d. Total %d\n", rowInsertedCount, duplicateCounter, rowCount)
}
