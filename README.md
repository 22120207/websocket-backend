# Example json data from client send to WebSocket server

```
websocat ws://localhost:8080/ws?target=14.225.204.41

# tcpdump -nni any tcp
{"type":"command","command":"dGNwZHVtcCAtbm5pIGFueSB0Y3AK"}

# journalctl -u webhooks-notify -f
{"type":"command","command":"am91cm5hbGN0bCAtdSB3ZWJob29rcy1ub3RpZnkgLWYK"}

# ls .
{"type":"command","command":"bHMgLgo="}
```