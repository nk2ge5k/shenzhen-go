
GO := go

go-update: # Update dependencies
	@$(GO) get $(shell $(GO) list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all)
.PHONY: go-update
