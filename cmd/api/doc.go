package main

// @title           Financial Wallet API
// @version         1.0
// @description     API para gerenciamento financeiro com Clean Architecture.

// @contact.name   Denyson Grellert
// @contact.url    https://github.com/DenysonJ/financial-wallet

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey ServiceName
// @in header
// @name Service-Name
// @description Name of the calling service (e.g. "billing-api")

// @securityDefinitions.apikey ServiceKey
// @in header
// @name Service-Key
// @description Secret key for the calling service

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT access token (format: "Bearer {token}")

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
