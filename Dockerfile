FROM golang:latest AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o main . 

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main /app/main 

COPY --from=builder /app/static /app/static

EXPOSE 3001

CMD ["./main"] 
