FROM node:24-alpine
WORKDIR /app
COPY node-services/package.json node-services/package-lock.json ./
COPY node-services/shared/ ./shared/
COPY node-services/auth-service/package.json ./auth-service/
COPY node-services/gen/ ./gen/
RUN cd /app && npm ci --omit=dev -w auth-service
WORKDIR /app/auth-service
COPY node-services/auth-service/tracing.js node-services/auth-service/server.js node-services/auth-service/app.js node-services/auth-service/handlers.js node-services/auth-service/consumer.js ./
COPY node-services/auth-service/migrations/ ./migrations/
EXPOSE 8090
CMD ["sh", "-c", "if [ -n \"$DATABASE_URL\" ]; then npx node-pg-migrate up --database-url $DATABASE_URL --migrations-dir ./migrations; fi && node --import ./tracing.js server.js"]
