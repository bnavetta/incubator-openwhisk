import socket
import sys
import time

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

def communicate(name, host, port):
	#'r name' --> registers name with ip address (no need to send ip address, server receives it upon connection)
	s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	print 'Socket created'
	print host
	print port
	s.connect((host, port))

	s.send(name)
	recv = s.recv(1024)
	print recv
	s.close()
	return 0



def main(args):
	# if ( (len(args) < 3) or (len(args) > 3) ) :
		# print "Usage: filename host port"
		# return

	global IP_HOST
	global IP_PORT
	
	IP_HOST = args['host']
	IP_PORT = int(args['port'])

	err = register('lambda_2')
	if (err):
		print "Error registering: Invalid Command"
		return
	print "Registered as lambda_2"

	addr, port = retrieve('lambda_1')
	while port == -1 :
		addr, port = retrieve('lambda_2')

	print "SUCCESS!"
	print "lambda_1 : ", addr, port

	time.sleep(7)

	communicate("hiiii\n", addr, port)
	communicate("how are u\n", addr, port)
	communicate("hello world!\n", addr, port)





if __name__ == "__main__":
	main()
