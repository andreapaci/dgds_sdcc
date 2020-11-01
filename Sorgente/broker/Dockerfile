# Si usa alpine come Parent Image per la sua leggerezza

FROM golang:1.15.3-alpine3.12
LABEL maintainer="Andrea Paci <andrea.paci1998@gmail.com>"

RUN mkdir /app
ADD . /app

ENV AWS_ACCESS_KEY_ID=
ENV AWS_SECRET_ACCESS_KEY=

WORKDIR /app
RUN apk update && \
    apk add git  && \
    go get github.com/aws/aws-sdk-go/aws/... && \
    go get github.com/aws/aws-sdk-go/service/dynamodb/... && \
    go get github.com/aws/aws-sdk-go/service/sqs/...

ADD common /go/src/common/
ADD github.com /go/src/github.com

EXPOSE 80
RUN go build -o broker
CMD ./broker	
