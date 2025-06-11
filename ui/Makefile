
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
	rm -rf bun.lock .next node_modules
	npm audit  || true
	npm update || true
	npm install
	bun install
	npm outdated || true
	@echo "Updating UI lock file done."

.PHONY: build
build:
	#npm install -D npm node eslint
	npm install geist
	rm -rf bun.lock
	bun run build --no-lint
	bun pm ls | tee | gzip | base64 -w0
	@echo "Bun dependencies hash:"
	bun pm hash

.PHONY: clean
clean:
	npm cache clean --force || true
	bun pm cache rm || true
	rm -rf .next
	rm -rf node_modules
	rm -rf bun.lock
	rm -rf dist
	rm -rf .bun
	rm -rf .cache

.PHONY: lint
lint:
	bun next lint

.PHONY: install-bun
install-bun:
	brew install bun

.PHONY: docker-build-strace
docker-build-strace:
	docker build -t kagents-ui-strace .
	docker create --name temp-kagents-ui-strace kagents-ui-strace
	docker cp temp-kagents-ui-strace:/app/ui/strace-logs ./strace-logs
	docker rm temp-kagents-ui-strace
	@echo "Strace logs copied to ./strace-logs directory"
