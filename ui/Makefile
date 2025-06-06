
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
	npm audit fix --force || true
	npm update    --force || true
	npm outdated || true
	npm install
	rm -rf bun.lock .next node_modules/.cache
	bun install
	@echo "Updating UI lock file done."

.PHONY: build
build: update
	bun run build --no-lint || bun run build -d 2>&1

.PHONY: clean
clean:
	rm -rf .next
	rm -rf node_modules
	rm -f bun.lock
	rm -rf dist
	rm -rf .bun
	rm -rf .cache

.PHONY: install-bun
install-bun:
	brew install bun