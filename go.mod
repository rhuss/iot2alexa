module github.com/rhuss/iot2alexa

go 1.13

require (
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/mikeflynn/go-alexa v0.0.0-20191016174603-1ffcf485965f
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.6.2
	github.com/urfave/negroni v1.0.0 // indirect
)

replace github.com/mikeflynn/go-alexa => github.com/rhuss/go-alexa v0.0.0-20200125232741-79f50bfa5aca
