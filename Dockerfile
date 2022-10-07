FROM golang:1.19.1

WORKDIR /usr/src/etl

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY *.go ./

RUN go build -v -o /usr/local/bin/etl
EXPOSE 3306

CMD [ "etl" ]
