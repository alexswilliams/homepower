FROM docker.io/golang:1.24.4-bookworm AS build
WORKDIR /go/src/app
COPY Makefile go.mod go.sum ./
RUN make deps
COPY . ./
RUN make bin/main

FROM scratch
WORKDIR /app
COPY --from=build /go/src/app/bin/main /app/main
ENTRYPOINT ["/app/main"]
