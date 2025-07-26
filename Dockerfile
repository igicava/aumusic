# Stage 1: Build
FROM golang:1.24-alpine3.21 AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod verify
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/main.go

# Stage 2: Image
FROM alpine:3.21

WORKDIR /app

COPY --from=build /app/app /app/app
COPY --from=build /app/.env /app/.env
COPY --from=build /app/db /app/db
COPY --from=build /app/frontend /app/frontend

EXPOSE 8081

ENTRYPOINT ["/app/app"]