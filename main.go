package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db *sql.DB
)

func main() {
	db, _ = sql.Open("sqlite3", "./database/database.db")
	go updateData()

	http.HandleFunc("/", getDataHandler)
	if err := http.ListenAndServe(":1234", nil); err != nil {
		panic(err)
	}
}

type request struct {
	Symbol         string  `json:"symbol" db:"symbol"`
	Price          float64 `json:"price_24h" db:"price_24h"`
	Volume         float64 `json:"volume_24h" db:"volume_24h"`
	LastTradePrice float64 `json:"last_trade_price" db:"last_trade_price"`
}

type resp struct {
	Price          float64 `json:"price"`
	Volume         float64 `json:"volume"`
	LastTradePrice float64 `json:"last_trade"`
}

func MakeRequest() ([]request, error) {
	resp, err := http.Get("https://api.blockchain.com/v3/exchange/tickers")
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var ourData []request
	if err := json.Unmarshal(body, &ourData); err != nil {
		return nil, err
	}

	return ourData, nil
}

func updateData() {
	for {
		data, err := MakeRequest()
		if err != nil {
			log.Println(err)
		}
		if err = saveData(db, data); err != nil {
			panic(err)
		}
		time.Sleep(30 * time.Second)
	}
}

func buildSQL(data []request) string {
	var saveDataSQL = `
INSERT INTO
	test
(
	symbol,
	price_24h,
	volume_24h,
	last_trade_price
) VALUES 
`
	for _, req := range data {
		saveDataSQL += fmt.Sprintf(`("%s", %f, %f, %f),`, req.Symbol, req.Price, req.Volume, req.LastTradePrice)
	}
	saveDataSQL = strings.TrimRight(saveDataSQL, ",")
	saveDataSQL += ";"
	return saveDataSQL
}

func saveData(db *sql.DB, data []request) error {
	query := buildSQL(data)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}

	return nil
}

const getDataSQL = `SELECT symbol, price_24h, volume_24h,last_trade_price FROM test;`

func getDataFromDB(db *sql.DB) (map[string]resp, error) {
	result := map[string]resp{}
	rows, err := db.Query(getDataSQL)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)
	for rows.Next() {
		key := ""
		rowData := resp{}
		err := rows.Scan(
			&key,
			&rowData.Price,
			&rowData.Volume,
			&rowData.LastTradePrice,
		)
		if err != nil {
			return nil, err
		}
		result[key] = rowData
	}
	return result, nil
}

func getDataHandler(w http.ResponseWriter, req *http.Request) {
	respData, err := getDataFromDB(db)
	if err != nil {
		panic(err)
	}
	jsonData, err := json.Marshal(respData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Fprint(w, string(jsonData))
}
