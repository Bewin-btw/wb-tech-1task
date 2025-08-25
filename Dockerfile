# builder
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/main ./cmd

# final (scratch)

FROM scratch

COPY --from=builder /app/main /main

COPY --from=builder /app/web /web

# если нужна работа с TLS и CA — см. вариант ниже

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/main"]
