package services

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"goParser/internal/config"
	"log"
)

type DbStorageStruct struct {
	Id                   string `db:"id"`
	LastUpdateInDb       string `db:"last_update_here"`
	LastUpdateFromSource string `db:"last_update_from_source"`
	Data                 string `db:"data"`
}

func InitDb(config config.DbConfigStruct, schema string) (*sqlx.DB, error) {
	//fmt.Println(config)
	db, err := sqlx.Connect(config.Driver, "user="+config.UserName+" password="+config.Password+" dbname="+config.DBName+" host="+config.DBHost+" port="+config.DBPort+" sslmode=disable")
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	db.MustExec(schema)

	return db, nil
}

func InsertToStorage(db *sqlx.DB, tableName string, needUpdateIfExists bool, dataForInsert []DbStorageStruct) (sql.Result, error) {

	sql := `INSERT INTO ` + tableName + ` (id,last_update_from_source,data) 
			VALUES (:id,:last_update_from_source,:data)`

	if needUpdateIfExists {
		sql += ` ON CONFLICT (id) DO UPDATE 
			  SET last_update_from_source = :last_update_from_source, 
				  data = :data`
	}

	result, err := db.NamedExec(sql, dataForInsert)
	//if err != nil {
	//	log.Fatalln(err)
	//}

	return result, err
}

func GetLastUpdateTime(db *sqlx.DB, tableName string) string {
	var lastTimeUpdate string
	err := db.Get(&lastTimeUpdate, "SELECT last_update_from_source FROM "+tableName+" ORDER BY last_update_from_source DESC LIMIT 1")
	if err != nil {
		return ""
	}

	return lastTimeUpdate
}

func CheckExistsById(db *sqlx.DB, tableName string, id string) bool {
	var amount int
	err := db.Get(&amount, "SELECT COUNT(*) AS amount FROM "+tableName+" WHERE id = $1", id)
	if err != nil || amount == 0 {
		return false
	}

	return true
}
