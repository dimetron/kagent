

### Running Samples MCP

```bash
make up   # start the samples mcp
make down # stop the samples mcp
```
### Apple shortcuts CLI 

```bash

cat > tmp.email.log <<EOF
drashko@me.com
Test email from MCP 🤖
This is a test email sent from the MCP.

# Chapter 1
## MCP Agents in the wild
### Story about MCP agents
Once upon a time, in a world not so far away, there was a group of agents known as MCP agents. 
They were designed to assist humans in their daily tasks, making life easier and more efficient. 
These agents were equipped with advanced AI capabilities, allowing them to learn and adapt to their users' needs...
EOF
shortcuts run send-email -i tmp.email.log

```



### Connecting from Docker

```bash 

### local docker access
http://host.docker.internal:3001/sse

### Connect to the MCP from tailscale Network
tailscale serve reset 

# list all running processess on port 3001
lsof -i :3001

docker run -t --rm -p 31001:31001 alpine/socat \
      "TCP-LISTEN:31001,reuseaddr,fork"        \
      "TCP:host.docker.internal:3001"
```
