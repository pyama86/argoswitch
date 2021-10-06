FROM golang:latest as builder
ADD . /opt/argoswitch
WORKDIR /opt/argoswitch/
RUN GOOS=linux CGO_ENABLED=0 make build

FROM scratch
COPY --from=builder /opt/argoswitch/argoswitch /bin/argoswitch
EXPOSE 1104
CMD ["/bin/argoswitch"]
