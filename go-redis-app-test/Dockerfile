FROM golang:latest as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download 
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o main .
# EXPOSE 80
# ENTRYPOINT ["./main"]


FROM gcr.io/distroless/base-debian11 
COPY --from=builder /app/main .
EXPOSE 80
CMD ["/main"]
  
#multistage build lightweight one redis password protect search
  