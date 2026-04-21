FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json ./
RUN npm install -g pnpm && pnpm install
COPY frontend/ ./
RUN pnpm build

FROM golang:1.23-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./public
RUN CGO_ENABLED=0 GOOS=linux go build -o jww .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-builder /app/jww .
COPY --from=backend-builder /app/public ./public
EXPOSE 3000
CMD ["./jww"]
