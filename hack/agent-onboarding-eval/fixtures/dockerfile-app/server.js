import {createServer} from 'node:http';

const port = Number(process.env.PORT ?? 3000);
const greeting = process.env.GREETING ?? 'Hello from Acme Reports';

createServer((req, res) => {
  res.writeHead(200, {'Content-Type': 'application/json'});
  res.end(JSON.stringify({service: 'acme-reports', greeting, path: req.url}));
}).listen(port, () => {
  console.log(`acme-reports listening on ${port}`);
});
