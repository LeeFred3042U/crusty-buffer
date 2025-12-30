APP_NAME=crusty
ENTRY_POINT=cmd/crusty/main.go

.PHONY: run build clean

# "make run" now explicitly calls the "server" command
run:
	@echo "Starting $(APP_NAME) server..."
	go run $(ENTRY_POINT) server

# Create a usable binary in a "bin" folder
build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/$(APP_NAME) $(ENTRY_POINT)

clean:
	rm -f bin/$(APP_NAME)