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
	"time"
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
	fmt.Println("******Hello! I'm parser serbiarus******")
	db, _ := services.InitDb(config.DbConfig(), config.DbSchema())
	defer db.Close()

	var cursor int = 0
	var rowInsertedCount int64 = 0
	var rowCount int64 = 0
	var goRoutineCounter int = 0
	var lastUpdateTime string = services.GetLastUpdateTime(db, "storage1")

exit:
	for true {
		// Запрос посылаем на сервер в однопотоке, чтобы не перегружать сеть и сервер
		fmt.Printf("Start processing page %d\n", cursor)
		var targetUrl string = "https://vmeste.info/api/trpc/announcement.list?batch=1&input="

		targetUrl = targetUrl + url.QueryEscape(`{"0":{"json":{"filters":{"groupId":"969ac93b-90f7-4273-a617-2e8ca21a2d92","isCommercial":false,"isNonCommercial":false,"payment":null,"format":null,"location":null,"rubric":null,"type":"all","searchSlug":"","isOnline":null,"locations":[],"rubrics":[],"inSearch":true,"suggesting":true},"limit":`+strconv.Itoa(parseLimit)+`,"cursor":`+strconv.Itoa(cursor)+`},"meta":{"values":{"filters.isOnline":["undefined"]}}}}`)
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
		goRoutineCounter++
		go func() {
			for _, item := range contentData.Data {
				rowCount++
				//fmt.Println(key, item["id"], item["updated"])
				jsonStr, _ := json.Marshal(item)
				insetResult, _ := services.InsertToStorage(db, "storage1", forceMode, []services.DbStorageStruct{
					{
						Id:                   item["id"].(string),
						LastUpdateInDb:       "",
						LastUpdateFromSource: item["updated"].(string),
						Data:                 string(jsonStr),
					},
				})

				if insetResult != nil {
					inserted, _ := insetResult.RowsAffected()
					rowInsertedCount += inserted
				}
			}

			goRoutineCounter--
		}()

		if cursor == contentData.NextCursor || contentData.NextCursor == 0 {
			break
		}

		cursor = contentData.NextCursor
	}

	fmt.Printf("Waiting... ")
	for goRoutineCounter > 0 {
		time.Sleep(1 * time.Second)
	}
	fmt.Printf("Stored %d records from %d\n", rowInsertedCount, rowCount)
}
