import os
import socket
import sys

mapping = {}

def main():
	PORT = int(os.environ['REGISTRY_PORT'])
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
		if not data:  
			conn.sendall("Invalid Command")
		if len(data) <= 3: #??  
			conn.sendall("Invalid Command")

		#Interpret Command
		if data[0] == 'r':
			#r name
			data = data.split() #should add error checks after this
			name = data[1]
			mapping[name] = addr
			print mapping[name]
		
		if data[0] == 'g':
			#g name
			data = data.split() #should add error checks after this 
			name = data[1]
			if name in mapping:
				send = str(mapping[name][0]) +  " " + str(mapping[name][1])
				conn.sendall(send)
			else:
				conn.sendall("Invalid Name")

		#Close Connection
		conn.close()

	#Close server
	s.close()

if __name__ == "__main__":
	main();

# 10.116.52.83 