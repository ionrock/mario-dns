
build:
	go build main.go api.go dns.go

run:
	./main -redis_addr $(redis_addr) -pass $(redis_pass) -master $(mario_master)