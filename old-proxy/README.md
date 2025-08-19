### Instalation


go mod init proxy
go get github.com/gin-contrib/cors
go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/swaggo/http-swagger
swag init
go build
