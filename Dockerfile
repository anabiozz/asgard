FROM golang:1.9

ADD . /go/src/github.com/anabiozz/asgard
ADD ./default-config.toml /go/bin/

RUN go get -u github.com/kardianos/govendor
RUN cd /go/src/github.com/anabiozz/asgard && govendor init && govendor add +external && govendor sync
RUN go install github.com/anabiozz/asgard/main

CMD [ "/go/bin/main", "/go/bin/default-config.toml" ]

