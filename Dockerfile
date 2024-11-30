FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY *.go ./
COPY templates ./templates
COPY static ./static

RUN go build -o main .

EXPOSE 3001

CMD ["./main"]