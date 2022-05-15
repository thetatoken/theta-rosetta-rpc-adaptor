# ------------------------------------------------------------------------------
# Build Theta
# ------------------------------------------------------------------------------
FROM golang:1.14.1 as service-builder

ENV GO111MODULE=on
ENV THETA_HOME=/go/src/github.com/thetatoken/theta

WORKDIR $THETA_HOME

RUN git clone --branch account_delta https://github.com/thetatoken/theta-protocol-ledger.git .

RUN make install

# ------------------------------------------------------------------------------
# Build Theta Rosetta
# ------------------------------------------------------------------------------
ENV ROSETTA_HOME=/go/src/github.com/thetatoken/theta-rosetta-rpc-adaptor

WORKDIR $ROSETTA_HOME

RUN git clone https://github.com/thetatoken/theta-rosetta-rpc-adaptor.git .

RUN make install

# ------------------------------------------------------------------------------
# Build final image
# ------------------------------------------------------------------------------
FROM ubuntu:20.04

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates
RUN apt-get -y install curl
RUN apt-get -y install wget

RUN mkdir -p /app \
  && chown -R nobody:nogroup /app \
  && mkdir -p /data \
  && chown -R nobody:nogroup /data

WORKDIR /app

COPY --from=service-builder /go/src/github.com/thetatoken/theta/integration ./integration/

RUN mkdir -p ./mainnet/walletnode

# Copy binary from theta-builder
COPY --from=service-builder /go/bin/theta /app/theta
COPY --from=service-builder /go/bin/thetacli /app/thetacli

# # Copy binary from rosetta-builder
COPY --from=service-builder /go/bin/theta-rosetta-rpc-adaptor /app/theta-rosetta-rpc-adaptor

# Install service start script
COPY --from=service-builder \
  /go/src/github.com/thetatoken/theta-rosetta-rpc-adaptor/run.sh \
  /app/run.sh

# Set permissions for everything added to /app
RUN chmod -R 755 /app/*

EXPOSE 8080
EXPOSE 15872
EXPOSE 16888
EXPOSE 21000
EXPOSE 30001

ENTRYPOINT [ "/app/run.sh" ]