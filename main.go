package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/blockloop/scan/v2"
	"github.com/gocolly/colly"
	_ "github.com/lib/pq"
)

type item struct {
	StoryURL  string
	Source    string
	comments  string
	CrawledAt time.Time
	Comments  string
	Title     string
}

type Source struct {
	Id        string
	From      string
	Url       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Scrape struct {
	Id       int32  `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Hps      string `json:"hps"`
	LastDate string `json:"lastDate"`
	From     string `json:"from"`
}

func initSql() *sql.DB {
	db, err := sql.Open("postgres", "postgres://haqqer:UClV0x2JZhOe@ep-lingering-snowflake-436548.ap-southeast-1.aws.neon.tech/neondb")
	if err != nil {
		panic(err.Error())
	}
	return db
}

func Handler(res http.ResponseWriter, req *http.Request) {
	db := initSql()
	sourceIndex := make(map[string]Source)
	sources := []Source{}

	results := []Scrape{}
	var index int32 = 0
	rows, err := db.Query(`SELECT "id", "from", "url", "createdAt", "updatedAt" FROM "Sources"`)
	if err != nil {
		fmt.Println("error rows")
		panic(err)
	}

	err = scan.Rows(&sources, rows)
	if err != nil {
		panic(err)
	}
	// Instantiate default collector
	c := colly.NewCollector(
		colly.Async(true),
	)
	for _, v := range sources {
		sourceIndex[v.Url] = v
		c.Visit(v.Url)
	}
	c.OnHTML(".Jasa_Konsultansi_Badan_Usaha_Non_Konstruksi", func(e *colly.HTMLElement) {
		index += 1
		temp := Scrape{}
		temp.Id = index
		temp.Title = e.DOM.Children().Find("a").Text()
		temp.Hps = e.ChildText("td.table-hps")
		temp.Type = "Jasa Konsultasi Badan Usaha non Konstruksi"
		temp.LastDate = e.ChildText("td.center")
		sourceFrom := sourceIndex[e.Request.URL.String()]
		temp.From = sourceFrom.From
		results = append(results, temp)
	})
	c.OnHTML(".Jasa_Lainnya", func(e *colly.HTMLElement) {
		index += 1
		temp := Scrape{}
		temp.Id = index
		temp.Title = e.DOM.Children().Find("a").Text()
		temp.Hps = e.ChildText("td.table-hps")
		temp.Type = "Jasa Lainnya"
		temp.LastDate = e.ChildText("td.center")
		sourceFrom := sourceIndex[e.Request.URL.String()]
		temp.From = sourceFrom.From
		results = append(results, temp)
	})
	c.Wait()

	jsonInBytes, err := json.Marshal(results)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonInBytes)
}
func main() {
	// start := time.Now()
	// elapsed := time.Since(start)

	// log.Printf("Binomial took %s", elapsed)
	http.HandleFunc("/", Handler)

	fmt.Println("server started at localhost:8000")
	http.ListenAndServe(":8000", nil)
}
