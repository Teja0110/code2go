# How to run
## Server

```bash
./server
```

## Client
go
```bash
./client
```

python
```bash
python ntp.py
```
The client output as follow:

```
Current: 10/23 20:19:05
Current: 10/23 20:19:06
Current: 10/23 20:19:07
Current: 10/23 20:19:08
Current: 10/23 20:19:09
Current: 10/23 20:19:10
RPC call cost 1.482382ms
DEBUG: synced
Current: 10/23 20:20:51
Current: 10/23 20:20:52
Current: 10/23 20:20:53
```

__Notice__
The address is set to ntp.newnius.com, you have to add `127.0.0.1<tab>ntp.newnius.com` in `/etc/hosts` or change the address to localhost in client.
