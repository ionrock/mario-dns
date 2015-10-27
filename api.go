package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

// Zone Struct
type Zone struct {
	Name string `json:"name"`
}

func http_handle_zone(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(req)
	zone_name := vars["name"]

	exists, err := client.SIsMember("zones", zone_name).Result()
	if err != nil {
		log.Println("Couldn't talk to redis to sismember")
	}

	if exists == false {
		res.WriteHeader(http.StatusNotFound)
		rawJSON := map[string]string{
			"error": "zone not found",
		}
		errorJSON, _ := json.Marshal(rawJSON)
		fmt.Fprint(res, string(errorJSON))
		return
	}

	rawrecords, _ := client.SMembers(zone_name).Result()

	switch req.Method {
	case "GET":
		log.Println(fmt.Sprintf("HTTP GET /zones/%s", zone_name))
		records := map[string][]string{
			"records": rawrecords,
		}
		outgoingJSON, err := json.Marshal(records)
		if err != nil {
			log.Println(err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(res, string(outgoingJSON))
	case "DELETE":
		log.Println(fmt.Sprintf("HTTP DELETE /zones/%s", zone_name))
		for _, record := range rawrecords {
			client.Del(record)
		}
		client.SRem("zones", zone_name)
	}
}

func http_handle_zones(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case "GET":
		log.Println("HTTP GET /zones/")
		rawzones, _ := client.SMembers("zones").Result()
		zones := map[string][]string{
			"zones": rawzones,
		}
		outgoingJSON, err := json.Marshal(zones)
		if err != nil {
			log.Println(err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(res, string(outgoingJSON))
	case "POST":
		zone := new(Zone)
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&zone)
		if err != nil {
			log.Println(err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println(fmt.Sprintf("HTTP POST /zones/ name: %s", zone.Name))
		exists, err := client.SIsMember("zones", zone.Name).Result()
		if err != nil {
			log.Println("Couldn't talk to redis to sismember")
		}

		if exists == true {
			errs := fmt.Sprintf("Duplicate zone %s", zone.Name)
			log.Println(errs)
			http.Error(res, errs, http.StatusConflict)
			return
		}

		Add_empty_zone(zone.Name)

		outgoingJSON, err := json.Marshal(zone)
		if err != nil {
			log.Println(err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusCreated)
		fmt.Fprint(res, string(outgoingJSON))
	}
}

func GetRouter() http.Handler {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/zones", http_handle_zones).Methods("GET", "POST")
	router.HandleFunc("/zones/{name}", http_handle_zone).Methods("GET", "DELETE")
	return router
}
