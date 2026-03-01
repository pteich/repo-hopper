.PHONY: build run app app-amd64 app-arm64 build-linux build-windows-amd64 build-windows-arm64 build-all install clean deps

# ── Dependencies ──────────────────────────────────────────────────────────────

# Install gogio tool (required for .app/.exe bundling)
deps:
	go install gioui.org/cmd/gogio@latest

# ── Development ───────────────────────────────────────────────────────────────

# Build a plain binary for the current OS/arch (no bundling)
build:
	go build -o repo-hopper ./cmd/desktop/

# Run directly for development
run:
	go run ./cmd/desktop/

# ── macOS builds (run on macOS) ───────────────────────────────────────────────

# Build macOS .app bundle for the current arch (auto-detected)
app: deps
	$(eval ARCH := $(shell go env GOARCH))
	gogio -target macos -arch $(ARCH) -icon appicon.png -o RepoHopper.app ./cmd/desktop/

app-amd64: deps
	gogio -target macos -arch amd64 -icon appicon.png -o RepoHopper-amd64.app ./cmd/desktop/

app-arm64: deps
	gogio -target macos -arch arm64 -icon appicon.png -o RepoHopper-arm64.app ./cmd/desktop/

# Install into /Applications (ad-hoc signed, no Apple certificate required)
install: app
	$(eval ARCH := $(shell go env GOARCH))
	rm -rf /Applications/RepoHopper.app
	# gogio nests the bundle — pick the right variant if present
	@if [ -d "RepoHopper.app/RepoHopper_$(ARCH).app" ]; then \
		cp -R RepoHopper.app/RepoHopper_$(ARCH).app /Applications/RepoHopper.app; \
	else \
		cp -R RepoHopper.app /Applications/RepoHopper.app; \
	fi
	xattr -cr /Applications/RepoHopper.app
	codesign --force --deep --sign - /Applications/RepoHopper.app

# ── Linux build (run on Linux) ────────────────────────────────────────────────
# Requires: libx11-dev libxkbcommon-dev libgl1-mesa-dev libvulkan-dev
#           libwayland-dev xorg-dev libglib2.0-dev libgtk-3-dev
#           libpango1.0-dev libcairo2-dev
build-linux: deps
	$(eval ARCH := $(shell go env GOARCH))
	gogio -target linux -arch $(ARCH) -icon appicon.png -o repo-hopper-linux-$(ARCH) ./cmd/desktop/

# ── Windows builds (run on Windows with GCC, e.g. via MSYS2/mingw-w64) ───────
# Requires: mingw-w64 or MSYS2 with CGO support
build-windows-amd64: deps
	GOARCH=amd64 GOOS=windows CGO_ENABLED=1 \
	gogio -target windows -arch amd64 -icon appicon.png -o repo-hopper-windows-amd64.exe ./cmd/desktop/

build-windows-arm64: deps
	GOARCH=arm64 GOOS=windows CGO_ENABLED=1 \
	gogio -target windows -arch arm64 -icon appicon.png -o repo-hopper-windows-arm64.exe ./cmd/desktop/

# ── Build all for the current platform ───────────────────────────────────────
build-all:
	@PLATFORM=$(shell go env GOOS); \
	if [ "$$PLATFORM" = "darwin" ]; then \
		$(MAKE) app-amd64 app-arm64; \
	elif [ "$$PLATFORM" = "linux" ]; then \
		$(MAKE) build-linux; \
	elif [ "$$PLATFORM" = "windows" ]; then \
		$(MAKE) build-windows-amd64; \
	fi

# ── Cleanup ───────────────────────────────────────────────────────────────────
clean:
	rm -f repo-hopper repo-hopper-linux-* repo-hopper-windows-*.exe
	rm -rf RepoHopper.app RepoHopper-amd64.app RepoHopper-arm64.app
