import socket
import sys

IP_HOST = ''
IP_PORT = 0

def retrieve(name):
	#'g name' --> gets the ip address for name, if name doesn't exist it receives error: Invalid name
	# Format of recv: "ipaddress port" (space separated)
	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	print 'Socket created'
	s.connect((IP_HOST,IP_PORT))

	s.send('g' + ' ' + name)

	recv = s.recv(1024)
	if (recv == "Invalid Name" or recv == "Invalid Command"):
		return recv, -1
	recv = recv.split()
	addr = recv[0]
	port = int(recv[1])
	s.close()

	#returns addr as string , port as int
	return addr, port

def register(name):
	#'r name' --> registers name with ip address (no need to send ip address, server receives it upon connection)
	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	print 'Socket created'
	s.connect((IP_HOST,IP_PORT))

	s.send('r' + ' ' + name)
	recv = s.recv(1024)
	if (recv == "Invalid Command"):
		return 1
	s.close()
	return 0


def main(args):
	# if ( (len(sys.argv) < 3) or (len(sys.argv) > 3) ) :
	# 	print "Usage: filename host port"
	# 	return

	# gif ( (len(args) < 2) or (len(args) > 3) ) :
		# print "Usage: filename host port"
		# return

	global IP_HOST
	global IP_PORT
	
	# IP_HOST = sys.argv[1]
	# IP_PORT = int(sys.argv[2])

	IP_HOST = args['host']
	IP_PORT = int(args['port'])

	err = register('lambda_1')
	if (err):
		print "Error registering: Invalid Command"
		return
	print "Registered as lambda_1"

	address, port = retrieve('lambda_2')
	while port == -1 :
		address, port = retrieve('lambda_2')

	print "SUCCESS!"
	print "lambda_2 : ", address, port

	my_address, my_port = retrieve('lambda_1')


	#Going to Listen:
	PORT = my_port
	HOST = ''
	print HOST + " : " + str(PORT)

	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	print 'Socket created'

	#Bind socket to local host and port
	try:
		s.bind((HOST, PORT))
	except socket.error as msg:
		print 'Bind failed. Error Code : ' + str(msg[0]) + ' Message ' + msg[1]
		sys.exit()

	print 'Socket bind complete'

	#Start listening on socket
	s.listen(10)
	print 'Socket now listening'
	
	#Talk to client
	while 1:
		#wait to accept a connection - blocking call
		conn, addr = s.accept()
		print 'Connected with ' + addr[0] + ':' + str(addr[1])

		#Receive Command
		data = conn.recv(1024)
		print data
		conn.send("WELL DONE!")

		#Close Connection
		conn.close()

	#Close server
	s.close()





if __name__ == "__main__":
	main()

