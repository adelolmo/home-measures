package main

import (
	"fmt"
	"log"
	"time"
	"flag"
	"os"
	"net/http"
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
	"github.com/gorilla/mux"
)

var db *sql.DB

func init() {
	var err error
	databaseFile := os.Getenv("HM_DB_FILE")
	fmt.Printf("HM_DB_FILE: %s\n", databaseFile)
	db, err = sql.Open("sqlite3", databaseFile)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}
	fmt.Println("You connected to your database.")

	sqlStmt := `
	create table if not exists measures (temperature integer not null , created_at TEXT not null);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		fmt.Errorf("%q: %s\n", err, sqlStmt)
	}
}

type Measures struct {
	Measures []Measure `json:"measures"`
}

type Measure struct {
	Temperature int `json:"temperature"`
	CreatedAt   string `json:"createdAt"`
}

func main() {
	host := flag.String("host", "localhost", "Host")
	port := flag.Int("port", 8080, "Listening port")
	flag.Parse()
	router := mux.NewRouter()
	router.HandleFunc("/measures", getMeasures).Methods("GET")
	router.HandleFunc("/measures", addMeasure).Methods("POST")

	p := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("Listening on %s...\n", p)
	log.Fatal(http.ListenAndServe(p, router))
}

func getMeasures(w http.ResponseWriter, req *http.Request) {
	fmt.Println("GET /measures")

	rows, err := db.Query("SELECT temperature, created_at FROM measures order by created_at desc")
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer rows.Close()

	measures := make([]Measure, 0)

	for rows.Next() {
		m := Measure{}
		err := rows.Scan(&m.Temperature, &m.CreatedAt)
		if err != nil {
			http.Error(w, "{\"message\":\"Not found\",\"code\":404}", 404)
			return
		}
		measures = append(measures,
			Measure{Temperature:m.Temperature,
				CreatedAt:m.CreatedAt})

	}
	if err = rows.Err(); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	json.NewEncoder(w).Encode(Measures{Measures:measures})
}

func addMeasure(w http.ResponseWriter, req *http.Request) {
	var m Measure;
	json.NewDecoder(req.Body).Decode(&m);

	stmt, err := db.Prepare("INSERT INTO measures(temperature, created_at) values(?, ?)")
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	res, err := stmt.Exec(m.Temperature, time.Now().Format("2006-01-02T15:04:05.511Z"))
	_, err = res.LastInsertId()
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}