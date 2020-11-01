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
    go get github.com/aws/aws-sdk-go/service/sqs/... && \
    go get -u github.com/gorilla/mux && \
    go get github.com/rs/cors	


ADD common /go/src/common/



RUN go build -o publisher
CMD ["./publisher", "n"]	
