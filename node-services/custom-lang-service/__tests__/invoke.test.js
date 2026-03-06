import request from 'supertest';
import { describe, expect, it } from 'vitest';
import app from '../app.js';

describe('POST /invoke', () => {
  it('returns 200 with greeting for normal name', async () => {
    const res = await request(app).post('/invoke').send({ name: 'Alice' });
    expect(res.status).toBe(200);
    expect(res.body.message).toBe('Hello Alice from custom-lang-service!');
  });

  it('returns 200 with default greeting when name is omitted', async () => {
    const res = await request(app).post('/invoke').send({});
    expect(res.status).toBe(200);
    expect(res.body.message).toBe('Hello World from custom-lang-service!');
  });

  it('returns 401 for "unauthorized"', async () => {
    const res = await request(app).post('/invoke').send({ name: 'unauthorized' });
    expect(res.status).toBe(401);
    expect(res.body.error).toBe('unauthorized');
  });

  it('returns 404 for "notfound"', async () => {
    const res = await request(app).post('/invoke').send({ name: 'notfound' });
    expect(res.status).toBe(404);
    expect(res.body.error).toBe('not found');
  });

  it('returns 429 for "ratelimit"', async () => {
    const res = await request(app).post('/invoke').send({ name: 'ratelimit' });
    expect(res.status).toBe(429);
    expect(res.body.error).toBe('rate limited');
  });

  it('returns 403 for "forbidden"', async () => {
    const res = await request(app).post('/invoke').send({ name: 'forbidden' });
    expect(res.status).toBe(403);
    expect(res.body.error).toBe('forbidden');
  });

  it('returns 409 for "conflict"', async () => {
    const res = await request(app).post('/invoke').send({ name: 'conflict' });
    expect(res.status).toBe(409);
    expect(res.body.error).toBe('conflict');
  });

  it('returns 503 for "unavailable"', async () => {
    const res = await request(app).post('/invoke').send({ name: 'unavailable' });
    expect(res.status).toBe(503);
    expect(res.body.error).toBe('service unavailable');
  });
});

describe('GET /healthz', () => {
  it('returns 200 when DB is not configured', async () => {
    const res = await request(app).get('/healthz');
    expect(res.status).toBe(200);
    expect(res.text).toBe('ok\n');
  });
});
