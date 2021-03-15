# This image is a microservice in golang for the Degree chaincode
FROM golang:1.16-alpine

WORKDIR /go/src/github.com/holzeis/daphnis
ENV GO111MODULE=on

COPY go.mod go.sum ./

RUN go mod download 

COPY . .

# Build application
RUN go build -o daphnis .

# # Production ready image
# # Pass the binary to the prod image
# FROM golang:1.16-alpine as prod

# COPY --from=build /go/src/github.com/holzeis/daphnis/daphnis /app/daphnis
# COPY --from=build /go/src/github.com/holzeis/daphnis/tmpl/ /app/daphnis

USER 1000

CMD ./daphnis

EXPOSE 8080