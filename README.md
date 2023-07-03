### Scrapper LPSE API

---

Golang service api to scrape data from LPSE websites, list of LPSE websites must to prepared before, currently we are using postgres database to store LPSE website url

### Prerequisite

    `Go >= 1.19`

### How to run

1. `git clone`
2. `cd scrapper-lpse-go/`
3. set `DATABASE_URL`, `REDIS_URL`, `PORT`
4. `go mod tidy`
5. `go run main.go`
