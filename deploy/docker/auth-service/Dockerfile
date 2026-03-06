FROM node:24-alpine
WORKDIR /app
COPY node-services/auth-service/package.json node-services/auth-service/package-lock.json ./
RUN npm ci --omit=dev
COPY node-services/auth-service/tracing.js node-services/auth-service/server.js node-services/auth-service/app.js node-services/auth-service/db.js ./
COPY node-services/auth-service/migrations/ ./migrations/
EXPOSE 8090
CMD ["sh", "-c", "if [ -n \"$DATABASE_URL\" ]; then npx node-pg-migrate up --database-url $DATABASE_URL --migrations-dir ./migrations; fi && node --import ./tracing.js server.js"]
