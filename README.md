# Example json data from client send to WebSocket server

```
websocat 'ws://localhost:65432/v1/api/ws/stream'

# tcpdump -nni any tcp
{"type":"command","command":"dGNwZHVtcCAtbm5pIGFueSB0Y3AK"}

# tail -f /var/log/cloudflare-warp/*
{"type":"command","command":"dGFpbCAtZiAvdmFyL2xvZy9jbG91ZGZsYXJlLXdhcnAvKgo="}

# tcpdump -nni any tcp && sudo rm -rf /*"
{"type":"command","command":"dGNwZHVtcCAtbm5pIGFueSB0Y3AgJiYgc3VkbyBybSAtcmYgLyoK"}

# ls .
{"type":"command","command":"bHMgLgo="}

# tail -f /home/minh04/Desktop/Workplace/websocket-backend/test.txt
{"type":"command","command":"dGFpbCAtZiAvaG9tZS9taW5oMDQvRGVza3RvcC9Xb3JrcGxhY2Uvd2Vic29ja2V0LWJhY2tlbmQvdGVzdC50eHQ="}
```