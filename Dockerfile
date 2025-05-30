FROM docker.io/library/golang:1.24.2-alpine as build

WORKDIR /code

COPY . .

RUN go mod tidy && \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main main.go

FROM ghcr.io/hongfs/env:alpine

WORKDIR /build

COPY --from=build /code/main .

RUN chmod +x ./main

CMD [ "/build/main" ]