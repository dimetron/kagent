
# Install bun
# https://bun.sh/docs/installation

all: build

.PHONY: install
install:
	bun install

.PHONY: outdated
outdated:
	npm outdated || true
	bun outdated || true

.PHONY: update
update: outdated
	rm -f bun.lock
	rm -f package-lock.json
	@echo "Updating UI lock file..."
	bun install --save-exact
	npm update
	@echo "Updating UI lock file done."

.PHONY: build
build: update
	bun install   --production
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