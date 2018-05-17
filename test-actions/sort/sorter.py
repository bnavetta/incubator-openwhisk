#!/usr/bin/python
import fcntl
import struct
import thread
import threading
import time
import socket
import sys
import httplib


REG_HOST = ''
REG_PORT = 0

FILE_HOST = ''
FILE_PORT = 0

MY_NAME = ''
MY_ADDR = ''
MY_PORT = 0

ack = 0
ack_lock = thread.allocate_lock()

sort_list = []
list_lock = thread.allocate_lock()

timestamps = {"file_system" : [] , "sort" : [] , "assigner_sorter" : []}

# https://stackoverflow.com/a/24196955/1725688
# this is strange and magical
def get_ip_address(ifname):
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    return socket.inet_ntoa(fcntl.ioctl(
        s.fileno(),
        0x8915,  # SIOCGIFADDR
        struct.pack('256s', ifname[:15])
    )[20:24])

def retrieve(name):
   #'g name' --> gets the ip address for name, if name doesn't exist it receives error: Invalid name
   # Format of recv: "ipaddress port" (space separated)
   # print "1"
   s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
   print 'Socket created'
   s.connect((REG_HOST,REG_PORT))
   # print "2"
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

def register(name, addr):
   #'r name' --> registers name with ip address
   s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
   print 'Socket created'
   s.connect((REG_HOST,REG_PORT))

   send = 'r' + ' ' + name + ' ' + addr
   s.send(send)
   data = s.recv(1024)
   print("Response from registry: '%s'" % data)
   if (data == "Invalid Command"):
      return 1
   s.close()
   print "connection closed" 
   return 0


def listener(name):
   addr = get_ip_address('eth1')
   print "Detected overlay IP address " + addr

   s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
   #Bind socket to local host and port
   try:
      s.bind((addr, 7171))
   except socket.error as msg:
      print 'Listener Bind failed. Error Code : ' + str(msg[0]) + ' Message ' + msg[1]
      sys.exit()

   #Start listening on socket
   s.listen(2)
   print 'Listener Socket now listening'

   #Registering Myself
   err = register(MY_NAME, addr)
   if (err):
      print "Error registering: %s" % err
      return
   print "Registered as " + MY_NAME + " at " + addr
   
   #Talk to client
   while True:
      print "Waiting to accept assigner connection"
      #wait to accept a connection - blocking call
      conn, addr = s.accept()
      print 'Listener Connected with ' + addr[0] + ':' + str(addr[1])

      #Spawn a client thread
      thread.start_new_thread( client , (conn, addr) )

   #Close server
   s.close()


def client(conn, addr):
   global ack
   global sort_list

   print ' Client Thread Running'

   data = '-'
   while data != None :
      # print data + " whut?"
      if data == "DONE!!!":
         print "here 3!!!!"
         break
      
      data = conn.recv(3)

      print data
      num = int(data)
      print num
      data = conn.recv(num)
      print data

      if (data == "DONE!!!"):
         ack_lock.acquire()
         ack += 1
         print "ack"
         print ack
         ack_lock.release()
         data = None

      else:

         start = time.time()
         list_lock.acquire()
         sort_list.append(data)
         sort_list.sort()
         list_lock.release()
         end = time.time()
         timestamps["sort"].append(end - start)

   #Close Connection
   conn.close()


def kill():
   time.sleep(50)
   print '{ "status": "timed out " }'
   sys.exit(0)


def main(args):
   # if ( (len(args) < 6) or (len(args) > 6) ) :
   #    print "Usage: registry_host : string , registry_port: num, file_host: string, file_port : num , name : string, assigners : num"
   #    return

   #args is a dictionary
   #{registry_host : string , registry_port: num, file_host: string, file_port : num , name : string , assigners : num}

   timeout_thread = threading.Thread(target=kill)
   timeout_thread.daemon = True
   timeout_thread.start()

   global REG_HOST
   global REG_PORT

   global FILE_HOST
   global FILE_PORT

   global MY_NAME
   global MY_ADDR
   global MY_PORT

   REG_HOST = args['registry_host']
   REG_PORT = int(args['registry_port'])
   FILE_HOST = args['file_host']
   FILE_PORT = int(args['file_port'])
   MY_NAME = args['name']
   num_assigners = args['assigners']

   print("sorter.py: key = %s, registry = %s:%d, sort server = %s:%d" % (MY_NAME, REG_HOST, REG_PORT, FILE_HOST, FILE_PORT))
   print("Expecting %d assigners" % num_assigners)


   # Create Listener Thread
   try:
      thread.start_new_thread( listener, ("wtv",) )
   except:
      print "Error: unable to start thread"

   #Wait for all assigners to be done
   global ack

   while (ack < num_assigners) :
      if (int(ack) == int(num_assigners) ):
         break
      pass

   #Send data to filesystem
   s = '\n'
   BODY = s.join(sort_list)
   conn = httplib.HTTPConnection( FILE_HOST , FILE_PORT)
   print conn

   start = time.time()
   conn.request("PUT", "/sort/outputs/" + MY_NAME , BODY)
   end = time.time()

   timestamps["file_system"].append(end - start)


   print sort_list
   print timestamps
   return timestamps

if __name__ == "__main__":
   main()



# protocol of assigner communication
# unregister
# assigner shouldn't crash if sorter isn't up
# timer in assigners, script to run assigners to test concurrency

# ~ fault tolerance 





# >>> import httplib
# >>> BODY = "***filecontents***"
# >>> conn = httplib.HTTPConnection("localhost", 8080)
# >>> conn.request("PUT", "/file", BODY)
# >>> response = conn.getresponse()
# >>> print response.status, response.reason
# 200, OK
