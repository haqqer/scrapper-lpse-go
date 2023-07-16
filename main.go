package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/blockloop/scan/v2"
	"github.com/go-redis/redis"
	"github.com/gocolly/colly"
	_ "github.com/lib/pq"
)

var ctx = context.Background()

type Source struct {
	Id        string
	From      string
	Url       string
	CreatedAt time.Time
	UpdatedAt time.Time
}
type LPSE struct {
	Id         int32  `json:"id"`
	Owner      string `json:"owner"`
	Type       string `json:"type"`
	Hps        int64  `json:"hps"`
	DeadlineAt string `json:"deadlineAt"`
	Title      string `json:"title"`
	Url        string `json:"url"`
}

var rgx = regexp.MustCompile("[^0-9]+")

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

func doScrape(rd *redis.Client, db *sql.DB) {
	var index int32 = 0
	sourceIndex := make(map[string]Source)
	results := []LPSE{}

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
		temp := LPSE{}
		temp.Id = index
		temp.Title = e.DOM.Children().Find("a").Text()
		hpsString := e.ChildText("td.table-hps")
		splited := strings.Split(hpsString, ",")
		hpsConverted := rgx.ReplaceAllString(splited[0], "")
		hpsNumber, err := strconv.ParseInt(hpsConverted, 10, 64)
		if err != nil {
			fmt.Println("error parsing, ", err.Error())
		}
		temp.Hps = hpsNumber
		temp.Type = "Jasa Konsultasi Badan Usaha non Konstruksi"
		temp.Url = fmt.Sprintf("%s://%s%s", e.Request.URL.Scheme, e.Request.URL.Host, e.ChildAttr("a", "href"))
		temp.DeadlineAt = e.ChildText("td.center")
		temp.Owner = sourceIndex[e.Request.URL.String()].From
		results = append(results, temp)
	})
	c.OnHTML(".Jasa_Lainnya", func(e *colly.HTMLElement) {
		index += 1
		temp := LPSE{}
		temp.Id = index
		temp.Title = e.DOM.Children().Find("a").Text()
		hpsString := e.ChildText("td.table-hps")
		splited := strings.Split(hpsString, ",")
		hpsConverted := rgx.ReplaceAllString(splited[0], "")
		hpsNumber, err := strconv.ParseInt(hpsConverted, 10, 64)
		if err != nil {
			fmt.Println("error parsing, ", err.Error())
		}
		temp.Hps = hpsNumber
		temp.Type = "Jasa Lainnya"
		temp.Url = fmt.Sprintf("%s://%s%s", e.Request.URL.Scheme, e.Request.URL.Host, e.ChildAttr("a", "href"))
		temp.DeadlineAt = e.ChildText("td.center")
		temp.Owner = sourceIndex[e.Request.URL.String()].From
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
	rd.Set("status", 0, 0)
	fmt.Println("Done")
}

func Scrapper(res http.ResponseWriter, req *http.Request) {
	start := time.Now()
	rd := initRedis()
	db := initSql()

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

	go doScrape(rd, db)

	jsonInBytes, err := json.Marshal(map[string]string{
		"status": "ok",
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	end := time.Since(start)
	log.Printf("Binomial took %s", end)
	res.Write(jsonInBytes)
}

func Data(res http.ResponseWriter, req *http.Request) {
	rd := initRedis()
	result, err := rd.Get("lpse").Bytes()
	// file, err := ioutil.ReadFile("lpse.json")
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(result)
}

func main() {
	port := os.Getenv("PORT")
	// start := time.Now()
	// elapsed := time.Since(start)

	// log.Printf("Binomial took %s", elapsed)
	http.HandleFunc("/scrapper", Scrapper)
	http.HandleFunc("/data", Data)

	fmt.Println("server started at localhost:" + port)
	http.ListenAndServe(":"+port, nil)
}
