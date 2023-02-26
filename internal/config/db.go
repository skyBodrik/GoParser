package config

import (
	"os"
)

type DbConfigStruct struct {
	Driver   string
	UserName string
	Password string
	DBName   string
	DBHost   string
	DBPort   string
}

func DbConfig() DbConfigStruct {
	return DbConfigStruct{
		Driver:   "postgres",
		UserName: os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DBName:   os.Getenv("POSTGRES_DB"),
		DBHost:   os.Getenv("POSTGRES_HOST"),
		DBPort:   os.Getenv("POSTGRES_PORT"),
	}
}

func DbSchema() string {
	return `
		CREATE TABLE IF NOT EXISTS storage1 (
			id varchar,
			last_update_here timestamp default current_timestamp,
			last_update_from_source timestamp default null,
			data json,
			CONSTRAINT storage1_pk PRIMARY KEY (id)
		);

		CREATE TABLE IF NOT EXISTS storage2 (
			id varchar,
			last_update_here timestamp default current_timestamp,
			last_update_from_source timestamp default null,
			data json,
			CONSTRAINT storage2_pk PRIMARY KEY (id)
		);
	`
}
