package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	UrlTG  = "https://api.telegram.org/bot"
	hiHelp = `Бот может:
-при запросе /help дать подсказку как с ним взаимодействовать
-при запросе погоды сказать температуру и скорость ветра на улице в Минске (остальные города в разработке)
-дать информации о книге или авторе`
	urlMeteoMinsk = "https://api.open-meteo.com/v1/forecast?current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m&latitude=53.9006&longitude=27.5590"
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

func SendMes(client *http.Client, url string, chat_id int, text string) {
	body := MessBody{
		ChatId: chat_id,
		Text:   text,
	}
	byteBody, err := json.Marshal(body)
	if err != nil {
		log.Println(err)
		return
	}
	send, err := client.Post(url, "application/json", bytes.NewBuffer(byteBody))
	if err != nil {
		log.Println(err)
		return
	}
	defer send.Body.Close()
	fmt.Println("Статус отправки:", send.StatusCode)
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
			var bodyMeteo MeteoResponse
			err = json.NewDecoder(askMeteo.Body).Decode(&bodyMeteo)
			if err != nil {
				fmt.Println("json Meteo error: ", err)
			}
			SendMes(client, UrlTG+tokenTG+"/sendMessage", m.From.Id, fmt.Sprintf("Температура воздуха в минске %.2f%s, сторость ветра %.2f%s", bodyMeteo.Current.Temperature, bodyMeteo.CurrentUnits.Temperature, bodyMeteo.Current.Wind, bodyMeteo.CurrentUnits.Wind))
			SendMes(client, UrlTG+tokenTG+"/sendMessage", m.From.Id, fmt.Sprintf("Текущее время %s по UTC", bodyMeteo.Current.Time))
		case strings.Contains(mesText, "/help"):
			SendMes(client, UrlTG+tokenTG+"/sendMessage", m.From.Id, hiHelp)
		}
	}
}
