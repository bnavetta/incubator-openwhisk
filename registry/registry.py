import socket
import sys

mapping = {}

def main():
	if ( (len(sys.argv) < 2) or (len(sys.argv) > 2) ) :
		print "Usage: filename port"
		return
	
	PORT = int(sys.argv[1])
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
		# print "accepting"
		#wait to accept a connection - blocking call
		conn, addr = s.accept()
		print 'Connected with ' + addr[0] + ':' + str(addr[1])

		#Receive Command
		data = conn.recv(1024)
		print "Received command: '%s'" % data
		if not data:
			print "ERROR: empty command from %s" % addr[0]
			conn.sendall("Invalid Command (empty)")
			continue
		if len(data) < 3: #??  
			print "ERROR: too-short command from %s" % addr[0]
			conn.sendall("Invalid Command (short)")
			continue

		#Interpret Command
		if data[0] == 'r':
			#r name
			data = data.split() #should add error checks after this
			name = data[1]
			address = data[2]
			mapping[name] = address
			print "Mapping %s to %s" % (name, address)
		elif data[0] == 'g':
			#g name
			data = data.split() #should add error checks after this 
			name = data[1]
			if name in mapping:
				conn.sendall(mapping[name])
			else:
				conn.sendall("Invalid Name")
		else:
			print "ERROR: unknown command '%s' from %s" % (data[0], addr[0])
			conn.sendall("Invalid Command (%s)" % data[0])

		# print "closed prev connection"
		#Close Connection
		conn.close()

	#Close server
	s.close()

if __name__ == "__main__":
	main();

# 10.116.52.83 