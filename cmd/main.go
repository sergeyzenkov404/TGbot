package main

import (
	"TGbot/pkg/api"
	"TGbot/pkg/db"
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	urlFL = "https://api.fantlab.ru/"
)

func main() {
	client := &http.Client{
		Timeout: 1000 * time.Second,
	}
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	tokenTG := os.Getenv("tg")

	PSQLsourse := os.Getenv("PSQLsourse")
	offset := 0
	//PSQLsourse = "postgres://router_go:root@127.0.0.1:5432/router_go"
	dbpool, err := pgxpool.New(context.Background(), PSQLsourse)
	if err != nil {
		log.Fatal(err)
		//panic(err)
	}
	defer dbpool.Close()

	dbChan := make(chan api.Message, 20)
	botChan := make(chan api.Message, 20)

	go api.WorkerBot(client, botChan)
	go db.WorkerDB(dbpool, dbChan)

	for {
		upd := api.GetUpdate(client, api.UrlTG+tokenTG+"/", offset)

		for _, update := range upd {
			if update.Message.Text != "" {
				dbChan <- update.Message
				botChan <- update.Message
			}
			offset = update.Update_id + 1
		}
	}
}
