import app from './app.js';
import { startPoller } from './outbox.js';

const port = process.env.PORT || 3000;

app.listen(port, () => {
  console.log(`custom-lang-service listening on :${port}`);
  startPoller(500);
});
