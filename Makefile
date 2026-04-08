build:
	go build -o bin/domain-manager ./cmd/server

run:
	go run ./cmd/server

css:
	npm run build:css

css-watch:
	npx tailwindcss -i ./tailwind.src.css -o ./assets/static/css/tailwind.css --watch

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/

.PHONY: build run css css-watch test lint clean
