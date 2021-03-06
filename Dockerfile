FROM golang:latest as builder
RUN apt update -qqy && apt install -qqy ca-certificates
COPY . /opt/argoswitch
WORKDIR /opt/argoswitch/
RUN GOOS=linux CGO_ENABLED=0 make build

FROM scratch
COPY --from=builder /opt/argoswitch/binary/argoswitch /bin/argoswitch
EXPOSE 1104
CMD ["/bin/argoswitch"]
