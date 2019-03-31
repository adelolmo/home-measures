package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
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
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}
}

type Measures struct {
	Measures []Measure `json:"measures"`
}

type Measure struct {
	Temperature int    `json:"temperature"`
	CreatedAt   string `json:"createdAt"`
}

func main() {
	host := flag.String("host", "0.0.0.0", "Host")
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

	measures := make([]Measure, 0)

	for rows.Next() {
		m := Measure{}
		err := rows.Scan(&m.Temperature, &m.CreatedAt)
		if err != nil {
			http.Error(w, "{\"message\":\"Not found\",\"code\":404}", 404)
			return
		}
		measures = append(measures,
			Measure{Temperature: m.Temperature,
				CreatedAt: m.CreatedAt})

	}
	if err = rows.Err(); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	if err = rows.Close(); err != nil {
		fmt.Printf("unable to close db rows")
	}

	if err = json.NewEncoder(w).Encode(Measures{Measures: measures}); err != nil {
		log.Fatal("cannot serialize measures list")
	}
}

func addMeasure(w http.ResponseWriter, req *http.Request) {
	var m Measure
	if err:=json.NewDecoder(req.Body).Decode(&m); err!=nil{
		http.Error(w, "cannot understand request", 400)
		return
	}

	stmt, err := db.Prepare("INSERT INTO measures(temperature, created_at) values(?, ?)")
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	res, err := stmt.Exec(m.Temperature, time.Now().Format("2006-01-02T15:04:05.511Z"))
	if err != nil {
		fmt.Println(err)
		http.Error(w, "cannot get measure from db", 500)
		return
	}
	_, err = res.LastInsertId()
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}