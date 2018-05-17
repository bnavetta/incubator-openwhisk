import socket
import sys
import requests
import string
import time
import httplib
import threading

REG_HOST = ''
REG_PORT = 0
FILE_HOST = ''
FILE_PORT = 0

alphabet = 'ab'#cdefghijklmnopqrstuvwxyz'
my_partition = ''
new_dict = {}
connect_dict = {}
num_letters = len(alphabet)

timestamps = {"file_system" : [] , "sort" : [] , "assigner_sorter" : []}

def register(name):
	#'r name' --> registers name with ip address (no need to send ip address, server receives it upon connection)
	#time.sleep(5)
	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	# print 'Socket created'
	print REG_HOST
	print REG_PORT
	s.connect((REG_HOST,REG_PORT))
	addr = get_ip_address('eth1')
	print "Detected overlay IP address " + addr
	s.send('r' + ' ' + name + ' ' + addr)
	recv = s.recv(1024)
	if (recv == "Invalid Command"):
		return 1
	print "recv: " + recv

	s.close()
	return 0


def retrieve(name):
	#'g name' --> gets the ip address for name, if name doesn't exist it receives error: Invalid name
	# Format of recv: "ipaddress"
	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	# print 'Socket created'
	s.connect((REG_HOST,REG_PORT))

	s.send('g' + ' ' + name)

	addr = s.recv(1024)
	if (addr == "Invalid Name" or addr == "Invalid Command"):
		return None
	s.close()

	#returns addr as string
	return addr


def register_and_retrieve_ip_for_letter(letter):
	#register with registry
	# err = register(letter)
	# if (err):
	# 	print "Error registering: Invalid Command"
	# 	return 
	# print "Registered as sorter:" + letter

	#retrieve assigned port from registry
	address = retrieve(letter)
	while address is None :
		print ("Looking up %s failed, trying again", letter)
		address = retrieve(letter)

	print "SUCCESS. Retrieved ip address for letter:" + letter + "!"

	#store ip_address in {letter:ip_add} dictionary


	return address

def get_letter_given_index(idx):
	return alphabet[idx]

def connect(host, port):
	for res in socket.getaddrinfo(host, port, socket.AF_UNSPEC, socket.SOCK_STREAM):
		af, socktype, proto, canonname, sa = res
		print "Trying af=%s, socktype=%s, proto=%s, canonname=%s, sa=%s" % res
		try:
			s = socket.socket(af, socktype, proto)
			s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
			print "Created socket..."
		except socket.error:
			s = None
			continue
		try:
			s.connect(sa)
			print "Connected to %s" % sa
		except socket.error:
			s.close()
			s = None
			continue
		break
	if s is None:
		raise IOError("Unable to connect to server")
	return s

def populate_alphabet_ip_and_connect():

	for x in range(0, num_letters):
		print "Looking up %s in the registry" % get_letter_given_index(x)
		address = register_and_retrieve_ip_for_letter(get_letter_given_index(x))
		print "Found %s for %s" % (address, get_letter_given_index(x))
		new_dict[get_letter_given_index(x)] = {address,7171}
		#print "new_dict:{" + get_letter_given_index(x)+ "," + new_dict[get_letter_given_index(x)+ "}";
		# s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
		# print 'Socket created for letter: '+ get_letter_given_index(x)
		# s.connect((address, 7171))
		s = connect(address, 7171)
		connect_dict[get_letter_given_index(x)] = s
		print "connect_dict:{" + get_letter_given_index(x)+ "," + str(connect_dict[get_letter_given_index(x)])+"}"
	
	return

def contact_fs_for_partition(aid):
	global FILE_HOST
	global FILE_PORT
	# print "contact_fs_for_partition called: aid" + str(aid)
	# r = requests.get("http://example.com/foo/bar")
	# print "r.status_code:"
	# print r.status_code
	# print "r.headers"
	# print r.headers
	# print "r.content"
	# print r.content
	# print "		make http request to filesystem"

	print "Requesting partition %d" % aid

	conn = httplib.HTTPConnection (FILE_HOST, FILE_PORT)
	myurl = "/sort/partitions/" +str(aid)

	start = time.time()
	conn.request("GET",myurl, "")
	content = conn.getresponse().read()
	end = time.time()
	timestamps["file_system"].append(end-start)
	
	print "partition content:" + content
	return content
	# return "arohi\nadam\nbuffalo\ncarson"
	# return "arohi\nadams"

def kill():
	time.sleep(50)
	print '{ "status": "timed out " }'
	sys.exit(0)

def main(args):
   # if ( (len(args) < 5) or (len(args) > 5) ) :
   #    print "Usage: registry_host : string , registry_port: num, file_host: string, file_port : num , id : string"
   #    return

	timeout_thread = threading.Thread(target=kill)
	timeout_thread.daemon = True
	timeout_thread.start()

	global REG_HOST
	global REG_PORT
	global FILE_HOST
	global FILE_PORT

    #args is a dictionary
    #{registry_host : string , registry_port: num, file_host: string, file_port : num , id : string}
	REG_HOST = args['registry_host']
	REG_PORT = args['registry_port']
	FILE_HOST = args['file_host']
	FILE_PORT = args['file_port']
	ASSIGNER_ID = args['id']

	print("assigner.py: id = %d, registry = %s:%d, sort server = %s:%d" % (ASSIGNER_ID, REG_HOST, REG_PORT, FILE_HOST, FILE_PORT))

	# REG_HOST = sys.argv[1]
	# REG_PORT = int(sys.argv[2])
	# FILE_HOST =  sys.argv[3]
	# FILE_PORT = int(sys.argv[4])
	# ASSIGNER_ID = int(sys.argv[5])

	#cslab7f:4343

	assigner_id = ASSIGNER_ID;

	#Get partition string
	my_partition = contact_fs_for_partition(assigner_id)
	print "partition_string: " + my_partition
	#Connect to sorters
	populate_alphabet_ip_and_connect()
	#Send words to sorters
	stripped_partition = my_partition.strip("\n")
	tokens = stripped_partition.split("\n")
	print tokens

	#Send each word to sorter
	for word in tokens:
		lower_word = word.lower()
		first_letter = lower_word[0]
		print "Sending %s to %s" % (lower_word, first_letter)
		s = connect_dict[first_letter]

		length = len(lower_word)
		print length
		print lower_word

		if length > 9 and length < 100:
			len_str = "0" + str(length)

		if length <= 9:
			len_str = "00" + str(length)

		if length > 99 and length < 1000:
			len_str = str(length)
		
		start = time.time()

		s.send(len_str) 
		s.send(lower_word)

		end = time.time()

		timestamps["assigner_sorter"].append(end-start)

		assigner_to_sorter_sendtime = end - start
		print assigner_to_sorter_sendtime
		#print lower_word

	#Send "ACK"s to sorters and close connection
	for x in range(0,num_letters):

		s = connect_dict[get_letter_given_index(x)]
		s.send("007") 
		s.send("DONE!!!")

		#sorter should recoginize "ACK" message
		s.close()
	print timestamps
	return timestamps


if __name__ == "__main__":
	main()
