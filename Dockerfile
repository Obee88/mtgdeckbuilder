# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:alpine AS backend-builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN go build -o server .

# Stage 3: Final image
FROM alpine:latest

WORKDIR /app
COPY --from=backend-builder /app/server .
COPY --from=frontend-builder /app/dist ./dist

EXPOSE 8090
CMD ["./server"]
