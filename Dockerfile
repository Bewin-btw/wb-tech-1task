FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/main ./cmd/app


FROM scratch

COPY --from=builder /app/main /main
COPY --from=builder /app/web /web
COPY --from=builder /app/migrations /migrations


USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/main"]
