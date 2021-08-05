FROM golang:1.14.2 as build

# RUN apt-get update && apt-get install -y curl make gcc g++ git
ENV GO111MODULE=on

# ENV GIT_TERMINAL_PROMPT=1

ENV THETA_TOKEN_HOME=$GOPATH/src/github.com/thetatoken

WORKDIR $THETA_TOKEN_HOME/theta
RUN git clone https://github.com/thetatoken/theta-protocol-ledger.git .

RUN make install
RUN cp -r ./integration/privatenet ../privatenet
RUN mkdir ~/.thetacli
RUN cp -r ./integration/privatenet/thetacli/* ~/.thetacli/
RUN chmod 700 ~/.thetacli/keys/encrypted

WORKDIR $THETA_TOKEN_HOME/theta-rosetta-rpc-adaptor
RUN git clone https://github.com/thetatoken/theta-rosetta-rpc-adaptor.git .

RUN make install

COPY ./run.sh $GOPATH/bin

# FROM alpine:latest
# RUN apk add --no-cache ca-certificates

ENV PATH=$GOPATH/bin:/usr/local/go/bin:/usr/local/bin:$PATH

RUN mkdir -p /app \
  && chown -R nobody:nogroup /app \
  && mkdir -p /data \
  && chown -R nobody:nogroup /data
RUN chmod -R 755 /app/*

CMD [ "run.sh" ]
EXPOSE 8080