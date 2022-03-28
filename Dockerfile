# Build image
FROM golang:1.18

LABEL description="dcrvotingweb build"
LABEL version="1.0"
LABEL maintainer "jholdstock@decred.org"

USER root
WORKDIR /root

COPY ./ /root/

RUN go build -v .

# Serve image
FROM golang:1.18

LABEL description="dcrvotingweb serve"
LABEL version="1.0"
LABEL maintainer "jholdstock@decred.org"

USER root
WORKDIR /root

COPY --from=0 /root/dcrvotingweb /root/
COPY --from=0 /root/public       /root/public

EXPOSE 8000
CMD ["/root/dcrvotingweb"]
