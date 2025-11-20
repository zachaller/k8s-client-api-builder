.PHONY: generate-parser test clean

generate-parser:
	@echo "Generating DSL parser from grammar..."
	cd pkg/dsl && goyacc -o parser_generated.go -p yy grammar.y
	@echo "✓ Parser generated successfully"

test:
	go test ./...

clean:
	rm -f pkg/dsl/parser_generated.go

.DEFAULT_GOAL := generate-parser
