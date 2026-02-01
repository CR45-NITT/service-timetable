FROM golang:1.21 AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o /out/service-timetable ./cmd/timetable

FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=builder /out/service-timetable /service-timetable

EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/service-timetable"]
