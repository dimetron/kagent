
# Install bun
# https://bun.sh/docs/installation

all: build

.PHONY: outdated
outdated:
	npm outdated || true

.PHONY: update
update: outdated
	@echo "Updating UI lock file..."
	npm update
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