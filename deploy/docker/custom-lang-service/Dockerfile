FROM node:24-alpine
WORKDIR /app
COPY node-services/package.json node-services/package-lock.json ./
COPY node-services/shared/ ./shared/
COPY node-services/custom-lang-service/package.json ./custom-lang-service/
RUN cd /app && npm ci --omit=dev -w custom-lang-service
WORKDIR /app/custom-lang-service
COPY node-services/custom-lang-service/tracing.js node-services/custom-lang-service/server.js node-services/custom-lang-service/app.js node-services/custom-lang-service/auth-middleware.js ./
COPY node-services/custom-lang-service/migrations/ ./migrations/
EXPOSE 3000
CMD ["sh", "-c", "if [ -n \"$DATABASE_URL\" ]; then npx node-pg-migrate up --database-url $DATABASE_URL --migrations-dir ./migrations; fi && node --import ./tracing.js server.js"]
