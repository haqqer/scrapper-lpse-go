package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/blockloop/scan/v2"
	"github.com/go-redis/redis"
	"github.com/gocolly/colly"
	_ "github.com/lib/pq"
)

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
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err.Error())
	}
	return db
}

func initRedis() *redis.Client {
	opt, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	client := redis.NewClient(opt)
	return client
}

func doScrape(rd *redis.Client, sources []Source) {
	var index int32 = 0
	sourceIndex := make(map[string]Source)
	results := []Scrape{}

	fmt.Println("Scrapping ... ")
	rd.Set("status", 1, 0)
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
		panic(err)
	}
	rd.Set("lpse", jsonInBytes, 0)
	// err = ioutil.WriteFile("lpse.json", jsonInBytes, os.ModePerm)
	// if err != nil {
	// 	panic(err)
	// }
	fmt.Println("Done")
	rd.Set("status", 0, 0)
}

func Scrapper(res http.ResponseWriter, req *http.Request) {
	rd := initRedis()
	db := initSql()
	sources := []Source{}

	rows, err := db.Query(`SELECT "id", "from", "url", "createdAt", "updatedAt" FROM "Sources"`)
	if err != nil {
		fmt.Println("error rows")
		panic(err)
	}

	err = scan.Rows(&sources, rows)
	if err != nil {
		panic(err)
	}

	res.Header().Set("Content-Type", "application/json")
	status := rd.Get("status").Val()
	if val, _ := strconv.Atoi(status); val == 1 {
		result, err := json.Marshal(map[string]string{
			"status": "processing...",
		})
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Write(result)
		return
	}

	go doScrape(rd, sources)

	jsonInBytes, err := json.Marshal(map[string]string{
		"status": "ok",
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	res.Write(jsonInBytes)
}
