FROM ghcr.io/hongfs/env:golang120 as build

WORKDIR /code

COPY . .

RUN go mod tidy && \
    env GOOS=linux GOARCH=amd64 go build -o main main.go

CMD ["./main"]