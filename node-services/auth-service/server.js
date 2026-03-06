import app from './app.js';

const port = process.env.PORT || 8090;

app.listen(port, () => {
  console.log(`auth-service listening on :${port}`);
});
