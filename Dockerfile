FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./

RUN if [ -f "go.mod" ]; then go mod tidy; fi

COPY . .

RUN go build -o main .

EXPOSE 8080

CMD ["/app/main"]
