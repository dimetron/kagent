up:
	#tailscale serve reset
	#tailscale serve --bg --https 3001 3001
	kubectl apply -f resources/agent.yaml -n kagent
	kubectl apply -f resources/agent.yaml -n kagent
	go build .
	./go-mcp-agent

down:
	#tailscale serve --http=3001 off
