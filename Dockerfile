# builder image
FROM golang:1.11

LABEL description="hardforkdemo"
LABEL version="1.0"
LABEL maintainer "holdstockjamie@gmail.com"

USER root
RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go build

EXPOSE 8000
CMD ["/app/hardforkdemo"]
