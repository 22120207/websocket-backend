import ws from 'k6/ws';
import { Rate } from 'k6/metrics';

export let wsErrorRate = new Rate('ws_error_rate');

export const options = {
  stages: [
    { duration: '10s', target: 20 },
    { duration: '20s', target: 40 },
    { duration: '30s', target: 100 },
    { duration: '20s', target: 0},
  ],
  thresholds: {
    ws_connecting: ['p(95)<300'], // 95 percent of connection times is below 300ms
    ws_error_rate: ['rate<0.1'], // less than 10% of connections should error
  },
};

export default function () {
  // const url = 'ws://14.225.250.154:65432/v1/api/ws/stream';
  const url = 'ws://localhost:8080/ws';

  const response = ws.connect(url, null, function (socket) {
    socket.on('open', function open() {
      // tcpdump -nni any tcp
      socket.send(JSON.stringify({ type: "command", command: "dGNwZHVtcCAtbm5pIGFueSB0Y3AK" }));

      wsErrorRate.add(0);

      socket.setInterval(function timeout() {
        socket.ping();
      }, 1000);
    });

    socket.on('error', (e) => {
        if (e.error() != 'websocket: close sent') {
            console.log('An unexpected error occurred: ', e.error());
        }
        wsErrorRate.add(1);
    });

    socket.setTimeout(function () {
        socket.close();
    }, 80 * 10**3);
  });
}