package handler

import (
	"net/http"
)

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
