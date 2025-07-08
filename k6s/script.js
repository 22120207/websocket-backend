import ws from 'k6/ws';
import { Rate } from 'k6/metrics';

export let wsErrorRate = new Rate('ws_error_rate');

export const options = {
  stages: [
    { duration: '1m', target: 20 },
    { duration: '2m', target: 40 },
    { duration: '1m', target: 100 },
    { duration: '3m', target: 200 },
    { duration: '1m', target: 0},
  ],
  thresholds: {
    ws_connecting: ['p(95)<250'], // 95 percent of connection times is below 100ms
    ws_error_rate: ['rate<0.05'], // less than 5% of connections should error
  },
};

export default function () {
  const url = 'ws://14.225.250.154:65432/v1/api/ws/stream';

  const response = ws.connect(url, null, function (socket) {
    socket.on('open', function open() {
      console.log('connected');

      // tcpdump -nni any tcp
      socket.send(JSON.stringify({ type: "command", command: "dGNwZHVtcCAtbm5pIGFueSB0Y3AK" }));

      socket.setInterval(function timeout() {
        socket.ping();
      }, 1000);
    });

    socket.on('close', () => console.log('disconnected'));

    socket.on('error', (e) => {
        if (e.error() != 'websocket: close sent') {
            console.log('An unexpected error occurred: ', e.error());
        }
        ErrorRate.add(1);
    });

    socket.setTimeout(function () {
        socket.close();
    }, 60 * 10**3);
  });
}