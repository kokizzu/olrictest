FROM golang:1.18

WORKDIR /app

#COPY . /app 
#RUN go mod tidy
#RUN go run main.go

COPY ./olrictest /app/

CMD [ '/app/olrictest' ]
