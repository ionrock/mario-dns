package main

import (
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"gopkg.in/redis.v3"
	"log"
	"net/http"
	"time"
)

var (
	client *redis.Client
	master string
)

func serve(net, ip, port string) {
	bind := fmt.Sprintf("%s:%s", ip, port)
	server := &dns.Server{Addr: bind, Net: net}

	dns.HandleFunc(".", handle)

	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("Failed to set up the "+net+"server %s", err.Error()))
	}
}

func prep_data() {
	Add_empty_zone("example.com")
	err := client.Set("example.com.1", "example.com. 3600 IN A 192.168.38.4", 0).Err()
	if err != nil {
		panic(err)
	}
	client.SAdd("example.com", "example.com.1")
}

func Add_empty_zone(zone_name string) {
	now := time.Now()
	serial := now.Unix()

	res, err := client.SAdd("zones", zone_name).Result()
	log.Println(fmt.Sprintf("Adding %s to zones, result: %s", zone_name, res))
	if err != nil {
		panic(err)
	}

	err = client.Set(zone_name+".2", zone_name+". 86400  IN  NS  ns.mariodns.com.", 0).Err()
	client.SAdd(zone_name, zone_name+".2")

	soa := fmt.Sprintf(zone_name+"  300  IN  SOA  ns.mariodns.com.  lol.mariodns.com. %d 10800 3600 604800 300", serial)
	err = client.Set(zone_name+".6", soa, 0).Err()
	client.SAdd(zone_name, zone_name+".6")
}

func main() {
	var redis_addr = flag.String("redis_addr", "127.0.0.1:6379", "redis ip:port")
	var pass = flag.String("pass", "", "password for redis, if necessary")
	var master_raw = flag.String("master", "127.0.0.1:53", "master for axfrs")
	var port = flag.String("port", "53", "port to run on")
	flag.Parse()

	master = *master_raw
	client = redis.NewClient(&redis.Options{
		Addr:     *redis_addr,
		Password: *pass,
		DB:       0,
	})

	pong, err := client.Ping().Result()
	log.Println(fmt.Sprintf("Ping to Redis result: %s, error: %s", pong, err))

	prep_data()

	go serve("udp", "0.0.0.0", *port)
	go serve("tcp", "0.0.0.0", *port)

	router := GetRouter()
	http.ListenAndServe(":8080", router)
}
