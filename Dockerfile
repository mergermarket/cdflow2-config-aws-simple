FROM golang:alpine AS build
WORKDIR /
RUN apk add -U ca-certificates git
ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM scratch
COPY --from=build /app /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
VOLUME /tmp
ENV TMPDIR /tmp

LABEL type="platform"

ENTRYPOINT ["/app"]
