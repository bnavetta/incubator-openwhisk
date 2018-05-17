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
	if (recv == "Invalid Name" || recv == "Invalid Command"):
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


def main():
	if ( (len(sys.argv) < 3) or (len(sys.argv) > 3) ) :
		print "Usage: filename host port"
		return

	global IP_HOST
	global IP_PORT
	
	IP_HOST = sys.argv[1]
	IP_PORT = int(sys.argv[2])

	err = register('sample')
	if (err):
		print "Error registering: Invalid Command"
		return

	addr, port = retrieve('sample')
	if (port < 0):
		print "Error retrieving: addr"
		return 

	print addr, port

if __name__ == "__main__":
	main()

