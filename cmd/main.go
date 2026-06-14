package main

import (
	"TGbot/pkg/api"
	"TGbot/pkg/db"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	urlMeteoMinsk = "https://api.open-meteo.com/v1/forecast?current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m&latitude=53.9006&longitude=27.5590"
	urlTG         = "https://api.telegram.org/bot"
	urlFL         = "https://api.fantlab.ru/"
	PSQLsourse    = "postgres://router_go:root@127.0.0.1:5432/router_go"
	hiHelp        = `Бот может:
-при запросе /help дать подсказку как с ним взаимодействовать
-при запросе погоды сказать температуру и скорость ветра на улице в Минске (остальные города в разработке)
-дать информации о книге или авторе`
)

func GetUpdate(client *http.Client, url string, offset int) []api.Update {
	reqUrl := fmt.Sprintf("%sgetUpdates?offset=%d&timeout=100", url, offset)
	mes, err := client.Get(reqUrl)
	if err != nil {
		log.Println("error update: ", err)
		return nil
	}

	defer mes.Body.Close()

	var resp api.TelegramResponse
	err = json.NewDecoder(mes.Body).Decode(&resp)
	if err != nil {
		log.Println("error decod: ", err)
		return nil
	}

	return resp.Result
}

func SendMes(client *http.Client, url string, chat_id int, text string) {
	body := api.MessBody{
		ChatId: chat_id,
		Text:   text,
	}
	byteBody, err := json.Marshal(body)
	if err != nil {
		log.Println(err)
		return // Вместо log.Fatal делаем return
	}
	send, err := client.Post(url, "application/json", bytes.NewBuffer(byteBody))
	if err != nil {
		log.Println(err)
		return // Вместо log.Fatal делаем return
	}
	defer send.Body.Close()
	fmt.Println("Статус отправки:", send.StatusCode)
}

func WorkerBot(client *http.Client, mes <-chan api.Message) {
	clientMeteo := &http.Client{
		Timeout: 1000 * time.Second,
	}
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	tokenTG := os.Getenv("tg")
	for m := range mes {
		mesText := strings.ToLower(m.Text)
		switch {
		case strings.Contains(mesText, "погода"):
			askMeteo, err := clientMeteo.Get(urlMeteoMinsk)
			if err != nil {
				fmt.Println("ask error: ", err)
			}
			var bodyMeteo api.MeteoResponse
			err = json.NewDecoder(askMeteo.Body).Decode(&bodyMeteo)
			if err != nil {
				fmt.Println("json Meteo error: ", err)
			}
			SendMes(client, urlTG+tokenTG+"/sendMessage", m.From.Id, fmt.Sprintf("Температура воздуха в минске %.2f%s, сторость ветра %.2f%s", bodyMeteo.Current.Temperature, bodyMeteo.CurrentUnits.Temperature, bodyMeteo.Current.Wind, bodyMeteo.CurrentUnits.Wind))
			SendMes(client, urlTG+tokenTG+"/sendMessage", m.From.Id, fmt.Sprintf("Текущее время %s по UTC", bodyMeteo.Current.Time))
		case strings.Contains(mesText, "/help"):
			SendMes(client, urlTG+tokenTG+"/sendMessage", m.From.Id, hiHelp)
		}
	}
}

func main() {
	client := &http.Client{
		Timeout: 1000 * time.Second,
	}
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	tokenTG := os.Getenv("tg")
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

	go WorkerBot(client, botChan)
	go db.WorkerDB(dbpool, dbChan)

	for {
		upd := GetUpdate(client, urlTG+tokenTG+"/", offset)

		for _, update := range upd {
			if update.Message.Text != "" {
				dbChan <- update.Message
				botChan <- update.Message
			}
			offset = update.Update_id + 1
		}
	}
}
