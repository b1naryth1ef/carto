FROM golang:1.24-alpine

RUN mkdir -p /usr/src/
WORKDIR /usr/src/

RUN apk add --no-cache --update curl gcc musl-dev

COPY go.mod go.sum /usr/src/carto/

WORKDIR /usr/src/carto
RUN go mod download

COPY . /usr/src/carto/

ENV CGO_ENABLED=0
RUN go build -v -o /bin/carto cmd/carto/main.go

ENTRYPOINT ["/bin/carto"]
