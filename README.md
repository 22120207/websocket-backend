# Example json data from client send to WebSocket server

```
websocat 'ws://localhost:8080/ws'

# tcpdump -nni any tcp
{"type":"command","command":"dGNwZHVtcCAtbm5pIGFueSB0Y3AK"}

# ls .
{"type":"command","command":"bHMgLgo="}

# tail -f /home/minh04/Desktop/Workplace/websocket-backend/test.txt
{"type":"command","command":"dGFpbCAtZiAvaG9tZS9taW5oMDQvRGVza3RvcC9Xb3JrcGxhY2Uvd2Vic29ja2V0LWJhY2tlbmQvdGVzdC50eHQ="}
```