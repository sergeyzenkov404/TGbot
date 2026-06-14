package main

import (
	"TGbot/pkg/api"
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
	urlTG         = "https://api.telegram.org/bot8713578224:"
	urlFL         = "https://api.fantlab.ru/"
	PSQLsourse    = "postgres://router_go:root@127.0.0.1:5432/router_go"
	hiHelp        = `Бот может:
-при запросе /help дать подсказку как с ним взаимодействовать
-при запросе погоды сказать температуру и скорость ветра на улице в Минске (остальные города в разработке)
-дать информации о книге или авторе`
)

type MessBody struct {
	ChatId int    `json:"chat_id"`
	Text   string `json:"text"`
}

type User struct {
	Id         int    `json:"id"`
	Username   string `json:"username"`
	First_name string `json:"first_name"`
	Is_premium bool   `json:"is_premium"`
}

type Message struct {
	Message_id int    `json:"message_id"`
	From       User   `json:"from"`
	Text       string `json:"text"`
	// Document       Document `json:"document"`
}

type Update struct {
	Update_id int     `json:"update_id"`
	Message   Message `json:"message"`
}
type TelegramResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

func GetUpdate(client *http.Client, url string, offset int) []Update {
	reqUrl := fmt.Sprintf("%sgetUpdates?offset=%d&timeout=100", url, offset)
	mes, err := client.Get(reqUrl)
	if err != nil {
		log.Println("error update: ", err)
		return nil
	}

	defer mes.Body.Close()

	var resp TelegramResponse
	err = json.NewDecoder(mes.Body).Decode(&resp)
	if err != nil {
		log.Println("error decod: ", err)
		return nil
	}

	return resp.Result
}

func SavMes(dbpool *pgxpool.Pool, mes Message) error {
	trans1, err := dbpool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("trans error %w", err)
	}
	defer trans1.Rollback(context.Background())
	userTr := `
		INSERT INTO tg_users (id, username, first_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) 
		DO UPDATE SET username = EXCLUDED.username, first_name = EXCLUDED.first_name;
	`
	mesTr := `
		INSERT INTO message_history (user_id, text_content)
		VALUES ($1, $2);
	`
	//log.Println(mes.From.First_name)
	_, err = trans1.Exec(context.Background(), userTr, mes.From.Id, mes.From.Username, mes.From.First_name)
	if err != nil {
		return fmt.Errorf("user error %w", err)
	}

	_, err = trans1.Exec(context.Background(), mesTr, mes.From.Id, mes.Text)
	if err != nil {
		return fmt.Errorf("text error %w", err)
	}
	err = trans1.Commit(context.Background())
	if err != nil {
		return fmt.Errorf("not commited %w", err)
	}
	return nil
}

func SendMes(client *http.Client, url string, chat_id int, text string) {
	body := MessBody{
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

func WorkerDB(dbpool *pgxpool.Pool, mes <-chan Message) {
	for m := range mes {
		err := SavMes(dbpool, m)
		if err != nil {
			fmt.Println("messege error: ", err)
		}
	}
}

func WorkerBot(client *http.Client, mes <-chan Message) {
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

	dbChan := make(chan Message, 20)
	botChan := make(chan Message, 20)

	go WorkerBot(client, botChan)
	go WorkerDB(dbpool, dbChan)

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
