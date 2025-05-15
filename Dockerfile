FROM golang:1.24.3-alpine AS builder

WORKDIR /app

COPY ./go.* .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /user-service ./cmd/app/main.go

FROM alpine

WORKDIR /app

COPY --from=builder /user-service .
COPY .env .env

EXPOSE 50050

CMD [ "./user-service", "--config=./.env" ]