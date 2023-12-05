FROM docker.io/golang:1.21.4-bullseye as build
WORKDIR /go/src/app
COPY . ./
RUN make bin/main

FROM scratch
WORKDIR /app
COPY --from=build /go/src/app/bin/main ./
ENTRYPOINT ["./main"]
