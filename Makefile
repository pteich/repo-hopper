.PHONY: build run app install clean deps

# Build the desktop binary
build:
	go build -o repo-hopper ./cmd/desktop/

# Run directly (for development)
run:
	go run ./cmd/desktop/

# Install gogio tool (required for macOS .app bundling)
deps:
	go install gioui.org/cmd/gogio@latest

# Build macOS .app bundle with icon
app: deps
	gogio -target macos -icon appicon.png -o RepoHopper.app ./cmd/desktop/

# Find architecture (usually amd64 or arm64), extract the valid bundle, remove quarantine, and ad-hoc sign it
install: app
	$(eval ARCH := $(shell go env GOARCH))
	rm -rf /Applications/RepoHopper.app
	cp -R RepoHopper.app/RepoHopper_$(ARCH).app /Applications/RepoHopper.app
	xattr -cr /Applications/RepoHopper.app
	codesign --force --deep --sign - /Applications/RepoHopper.app
# Clean build artifacts
clean:
	rm -f repo-hopper
	rm -rf RepoHopper.app
