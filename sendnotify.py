import dns.message
import dns.rdatatype
import dns.opcode
import dns.flags
import dns.query
import sys

zone = sys.argv[1]
notify = dns.message.make_query(zone, dns.rdatatype.SOA)
notify.set_opcode(dns.opcode.NOTIFY)
notify.flags -= dns.flags.RD

response = dns.query.udp(notify, '127.0.0.1', port=5358, timeout=5)
