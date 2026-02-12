FROM golang:1.25.6 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o /out/service-timetable ./cmd/timetable

FROM gcr.io/distroless/base-debian12

WORKDIR /

COPY --from=builder /out/service-timetable /service-timetable

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/service-timetable"]
