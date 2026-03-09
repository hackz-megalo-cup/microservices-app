FROM node:24-alpine
WORKDIR /app
COPY node-services/custom-lang-service/package.json node-services/custom-lang-service/package-lock.json ./
RUN npm ci --omit=dev
COPY node-services/custom-lang-service/tracing.js node-services/custom-lang-service/server.js node-services/custom-lang-service/app.js node-services/custom-lang-service/db.js node-services/custom-lang-service/auth-middleware.js node-services/custom-lang-service/idempotency.js node-services/custom-lang-service/kafka.js ./
COPY node-services/custom-lang-service/migrations/ ./migrations/
EXPOSE 3000
CMD ["sh", "-c", "if [ -n \"$DATABASE_URL\" ]; then npx node-pg-migrate up --database-url $DATABASE_URL --migrations-dir ./migrations; fi && node --import ./tracing.js server.js"]
