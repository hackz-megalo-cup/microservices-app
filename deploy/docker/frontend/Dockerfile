FROM node:24-alpine AS builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
ARG VITE_API_BASE_URL
ARG VITE_GAME_SERVER_URL
ARG VITE_OTEL_ENDPOINT
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_GAME_SERVER_URL=${VITE_GAME_SERVER_URL}
ENV VITE_OTEL_ENDPOINT=${VITE_OTEL_ENDPOINT}
RUN npm run build

FROM nginx:1.28-alpine
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
