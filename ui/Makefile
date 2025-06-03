
# Install bun
# https://bun.sh/docs/installation

all: build

.PHONY: install
install:
	bun install

.PHONY: outdated
outdated:
	npm outdated || true
	#bun outdated || true

.PHONY: update
update: outdated
#	@echo "Updating UI lock file..."
#	bun install
#	npm update
#	@echo "Updating UI lock file done."

.PHONY: build
build: update
	rm -f bun.lock
	bun install   --production
	bun run build --no-lint 2>&1

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