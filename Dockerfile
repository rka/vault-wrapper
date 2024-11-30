FROM golang:latest

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY templates ./templates
COPY static ./static

RUN go build -o main .

EXPOSE 3001

CMD ["./main"]