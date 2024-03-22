FROM golang:1.21

USER 0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY *.yml ./
COPY website/ ./website/
COPY static/ ./static/

RUN GOOS=linux go build -o /serve

EXPOSE 8000

ENV DOCKERIZE_VERSION v0.7.0
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

CMD dockerize -wait tcp://db:3306 -timeout 15s /serve /app/config.yml
