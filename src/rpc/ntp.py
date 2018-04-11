# -*- coding: utf-8 -*-
import grpc
import ntp_pb2
import datetime, time, threading

import ntp
ntp.delay = 0.0

running = True

def sync():
  try:
    creds = grpc.ssl_channel_credentials(open('server-cert.pem').read())
    channel = grpc.secure_channel('ntp.newnius.com:8844', creds)

    #channel = grpc.insecure_channel('ntp.newnius.com:8844')

    stub = ntp_pb2.NtpStub(channel)

    request = ntp_pb2.NtpRequest()
    while running:
      start = round(time.time() * 1000)
      #print start / 1000
      #time.sleep(5)
      reply = stub.Query(request)
      #time.sleep(5)
      m = ( round(time.time() * 1000) - start ) / 2
      print "RPC call used: ", m * 2, "ms"
      #print start
      #print reply.message / float(1000000)
      #m = 0
      ntp.delay = ( reply.message / float(1000000) - start - m ) / float(1000)
      #print ntp.delay
      print "DEBUG: synced"
      time.sleep(5)
  except Exception as e:
    print "error occured: ", e

def tick():
  while True:
    current = datetime.datetime.now() + datetime.timedelta(0, ntp.delay)
    print "Current: ", current
    time.sleep(1)

if __name__ == '__main__':
  try:
    #sync()
    t = threading.Thread(target=sync)
    t.start()
    tick()
  except KeyboardInterrupt:
    running = False
