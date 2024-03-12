package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"database/sql"

	_ "github.com/lib/pq"
)

func main() {
	db_url := "postgresql://scipex:scipex@192.168.0.52/country_thing?sslmode=disable"
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	db.SetMaxOpenConns(20)

	codes := generateCodes()
	for _, code := range codes {
		getPlayers(code, db)
	}
}

func getPlayers(code string, db *sql.DB) error {
	fmt.Println("Getting players for", code)
	url := "https://api.chess.com/pub/country/" + code + "/players"

	res, err := http.Get(url)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	resp := PlayerResp{}
	err = json.Unmarshal([]byte(body), &resp)
	if err != nil {
		return err
	}

	fmt.Println("Got", len(resp.Players), "players for", code)

	wg := sync.WaitGroup{}
	for i, player := range resp.Players {
		go inserPlayer(player, code, db, &wg, fmt.Sprint(i, "/", len(resp.Players)))
	}

	wg.Wait()

	return nil
}

func inserPlayer(
	player string,
	code string,
	db *sql.DB,
	wg *sync.WaitGroup,
	id string,
) {
	wg.Add(1)

	fmt.Println("Inserting", player, "into", code)

	exists, err := db.Query("SELECT 1 FROM entry WHERE username = $1", player)
	if err != nil {
		fmt.Println("Error? :", err)
	}
	if exists.Next() {
		exists.Close()

		fmt.Println("Player exists, updating country.", id)
		_, err = db.Exec("UPDATE entry SET country = $1 WHERE username = $2", code, player)
		if err != nil {
			fmt.Println("Error:", err)
		}
	} else {
		exists.Close()

		fmt.Println("Player does not exist, inserting.", id)
		_, err = db.Exec("INSERT INTO entry (username, country) VALUES ($1, $2)", player, code)
		if err != nil {
			fmt.Println("Error:", err)
		}
	}

	wg.Done()
}

type PlayerResp struct {
	Players []string `json:"players"`
}

func generateCodes() []string {
	result := make([]string, 0)
	for i := 'A'; i <= 'Z'; i++ {
		for j := 'A'; j <= 'Z'; j++ {
			result = append(result, string(i)+string(j))
		}
	}
	return result
}
