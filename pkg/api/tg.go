package api

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
