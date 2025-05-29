
# Install bun
# https://bun.sh/docs/installation

all: build

.PHONY: update
update:
	@echo "Updating UI lock file..."
	rm -f package-lock.json
	bun install
	bun update
	@echo "Updating UI lock file done."

.PHONY: build
build: update
	rm -f bun.lockb
	bun install   --frozen-lockfile --production
	bun run build --no-lint

.PHONY: clean
clean:
	rm -rf node_modules
	rm -f bun.lockb
	rm -rf dist
	rm -rf .bun
	rm -rf .cache

.PHONY: install-bun
install-bun:
	brew install bun