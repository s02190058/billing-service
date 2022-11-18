# step 1 - loading dependences
FROM golang:1.19.3-alpine AS modules
WORKDIR /modules
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# step 2 - compiling the source code into binary
FROM golang:1.19.3-alpine AS build
COPY --from=modules /go/pkg /go/pkg
WORKDIR /app
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
COPY go.mod ./
COPY go.sum ./
RUN CGO_ENABLED=0 GOOS=linux \
    go build -o /bin/app ./cmd/app

# step 3 - running binary application
FROM scratch
COPY --from=build /bin/app /app
COPY configs /configs
EXPOSE 8080
CMD ["/app"]