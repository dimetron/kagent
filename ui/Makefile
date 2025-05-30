
# Install bun
# https://bun.sh/docs/installation

all: build

.PHONY: install
install:
	rm -f package-lock.json
	bun install nextjs
	bun install node
	bun install npm
	bun install

.PHONY: outdated
outdated:
	bun install --lock-frozen
	bun outdated || true

.PHONY: update
update: outdated
	@echo "Updating UI lock file..."
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