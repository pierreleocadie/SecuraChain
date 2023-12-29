# List of applications (assuming each folder inside cmd/ is a separate app)
APPS := $(notdir $(wildcard ../cmd/*))
GOBUILD := CGO_ENABLED=0 go build

# The app you want to build (passed as an argument to make)
APP :=

# Determine the targets based on APP
ifneq ($(findstring $(APP),$(APPS)),)
# If APP is part of APPS, then just use APPS as the target
TARGETS = $(APPS)
else
# Otherwise, add APP to the list of targets
TARGETS = $(APP) $(APPS)
endif

# Build targets
.PHONY: run build-all build lint secure tests all-check clean all

run:
	mkdir -p ../temp
	@echo "Running $(APP)..."
	@GOLOG_FILE="../temp/`date '+%F-%H%M%S'`.log" \
	GOLOG_OUTPUT="stdout+file" \
	GOLOG_LOG_FMT="nocolor" \
	go run ../cmd/$(APP)/main.go

build-all: $(APPS)

build: $(APP)

$(TARGETS):
	# Linux
	mkdir -p ../bin/$@/linux_amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o ../bin/$@/linux_amd64/$@ ../cmd/$@/*.go
	mkdir -p ../bin/$@/linux_arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o ../bin/$@/linux_arm64/$@ ../cmd/$@/*.go
	# Windows
	mkdir -p ../bin/$@/windows_amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o ../bin/$@/windows_amd64/$@.exe ../cmd/$@/*.go
	mkdir -p ../bin/$@/windows_arm64
	GOOS=windows GOARCH=arm64 $(GOBUILD) -o ../bin/$@/windows_arm64/$@.exe ../cmd/$@/*.go
	# MacOS
	mkdir -p ../bin/$@/darwin_amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o ../bin/$@/darwin_amd64/$@ ../cmd/$@/*.go
	mkdir -p ../bin/$@/darwin_arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o ../bin/$@/darwin_arm64/$@ ../cmd/$@/*.go

lint:
	golangci-lint run ../...

secure:
	gosec -quiet ../...

tests:
	mkdir -p ../temp
	go test -v ../... > ../temp/tests_output_$(shell date +'%d_%m_%Y_%H_%M_%S').log 2>&1

all-check: lint secure tests

clean:
	go clean
	rm -rf ../bin ../temp

all: all-check build-all