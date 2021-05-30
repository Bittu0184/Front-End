FROM golang:latest

LABEL maintainer="Brajesh Mahajan <brajeshmahajan184@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main .

EXPOSE 3001

CMD ["./main"]