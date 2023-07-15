FROM golang:1.20.3
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main ./cmd
EXPOSE 5000
CMD ["./main"]
