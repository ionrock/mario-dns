package main

import (
	"fmt"
	"github.com/miekg/dns"
	"gopkg.in/redis.v3"
	"strconv"
	"strings"
	"time"
)

func handle(writer dns.ResponseWriter, request *dns.Msg) {
	message := new(dns.Msg)
	message.SetReply(request)
	message.SetRcode(message, dns.RcodeSuccess)
	question := request.Question[0]

	switch request.Opcode {
	case dns.OpcodeNotify:
        fmt.Println(fmt.Sprintf("Recieved NOTIFY for %s", question.Name))
		message = handle_notify(question, message, writer)
	case dns.OpcodeQuery:
        fmt.Println(fmt.Sprintf("Recieved QUERY for %s", question.Name))
		message = handle_query(question, message, writer)
	default:
		message = handle_error(message, writer, "REFUSED")
	}

	// Apparently this dns library takes the question out on
	// certain RCodes, like REFUSED, which is not right. So we reinsert it.
	message.Question[0].Name = question.Name
	message.Question[0].Qtype = question.Qtype
	message.Question[0].Qclass = question.Qclass
	message.MsgHdr.Opcode = request.Opcode

	// Send an authoritative answer
	message.MsgHdr.Authoritative = true

	writer.WriteMsg(message)
}

func handle_error(message *dns.Msg, writer dns.ResponseWriter, op string) *dns.Msg {
	switch op {
	case "REFUSED":
		message.SetRcode(message, dns.RcodeRefused)
	case "SERVFAIL":
		message.SetRcode(message, dns.RcodeServerFailure)
	default:
		message.SetRcode(message, dns.RcodeServerFailure)
	}

	return message
}

func handle_query(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	key := fmt.Sprintf("%s%d", question.Name, question.Qtype)
	switch question.Qtype {
	case 255:
		// TODO: handle ANY
	default:

	}
	// TODO: Handle multiple records at the same name

	val, err := client.Get(key).Result()
	if err == redis.Nil {
		message.SetRcode(message, dns.RcodeNameError)
	} else if err != nil {
		message.SetRcode(message, dns.RcodeServerFailure)
	} else {
		// If this isn't speedy, can fix later
		ansRR, _ := dns.NewRR(val)
		message.Answer = append(message.Answer, ansRR)
		fmt.Println(key, val)
	}

	return message
}

func handle_notify(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name
	// serial := get_serial(zone_name, "127.0.0.1:53")
	// if serial == 0 {
	//     return handle_error(message, writer, "SERVFAIL")
	// }

	// Check our master for the SOA of this zone
	// master_serial := get_serial(zone_name, master)
	// if master_serial == 0 {
	//     // logger.Error(fmt.Sprintf("UPDATE ERROR %s : problem with master SOA query", zone_name))
	//     return handle_error(message, writer, "SERVFAIL")
	// }
	// if master_serial <= serial {
	//     // logger.Info(fmt.Sprintf("UPDATE SUCCESS %s : already have latest version %d", zone_name, serial))
	//     return message
	// }
	zone, err := do_axfr(zone_name)
	if len(zone) == 0 || err != nil {
        fmt.Println("There was a problem with the AXFR, or there were no records in it")
		return handle_error(message, writer, "SERVFAIL")
	}

	naive_update(strings.TrimSuffix(zone_name, "."), zone)

	return message
}

func naive_update(zone_name string, records []dns.RR) {
	rrs_raw := []string{}
	rrs_short := []string{}
	for _, rr := range records {
		rrs_raw = append(rrs_raw, strings.Replace(rr.String(), "\t", " ", 0))
		fmt.Println(rr.String())
		hdr := rr.Header()
		rrs_short = append(rrs_short, hdr.Name+strconv.Itoa(int(hdr.Rrtype)))
	}

	rawrecords, _ := client.SMembers(zone_name).Result()
	// Naively delete all records
	for _, record := range rawrecords {
		fmt.Println(fmt.Sprintf("deleting record %s", record))
		client.Del(record)
	}

	// Naively recreate all records from AXFR
	for i, rr_key := range rrs_short {
		fmt.Println(fmt.Sprintf("adding record %s", rr_key))
		client.Set(rr_key, rrs_raw[i], 0)
		client.SAdd(zone_name, rr_key)
	}
}

func get_serial(zone_name, query_dest string) uint32 {
	var serial uint32 = 0
	var in *dns.Msg

	m := new(dns.Msg)
	m.SetQuestion(zone_name, dns.TypeSOA)

	c := &dns.Client{DialTimeout: 10 * time.Second, ReadTimeout: 10 * time.Second}

	// _ is query time, might be useful later
	var err error
	in, _, err = c.Exchange(m, query_dest)
	if err != nil {
		return serial
	}
	return serial_query_parse(in)
}

func serial_query_parse(in *dns.Msg) uint32 {
	var serial uint32 = 0
	if in.Rcode != dns.RcodeSuccess {
		return serial
	}
	if rr, ok := in.Answer[0].(*dns.SOA); ok {
		serial = rr.Serial
	}
	return serial
}

func do_axfr(zone_name string) ([]dns.RR, error) {
	result := []dns.RR{}
	message := new(dns.Msg)
	message.SetAxfr(zone_name)
	transfer := &dns.Transfer{DialTimeout: 10 * time.Second, ReadTimeout: 10 * time.Second}

	channel, err := transfer.In(message, master)
	if err != nil {
		return result, err
	}

	for envelope := range channel {
		result = append(result, envelope.RR...)
	}
	return result, nil
}
