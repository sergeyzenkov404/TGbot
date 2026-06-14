package db

import (
	"TGbot/pkg/api"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func WorkerDB(dbpool *pgxpool.Pool, mes <-chan api.Message) {
	for m := range mes {
		err := SavMes(dbpool, m)
		if err != nil {
			fmt.Println("messege error: ", err)
		}
	}
}

func SavMes(dbpool *pgxpool.Pool, mes api.Message) error {
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
