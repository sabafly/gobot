FROM golang:1.25.4

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN go generate ./...

COPY . .
RUN go build -ldflags="-s -w" -o gobot .

CMD ["./gobot", "bot", "--pprof", "--debug"]
