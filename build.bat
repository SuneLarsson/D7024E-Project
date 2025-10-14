@echo off
REM This batch file automates the process of removing and redeploying a Docker stack.

echo --- Starting Docker Stack Redeployment ---

echo Step 1: Removing old stack 'kademlia-app'...
docker stack rm kademlia-app

echo Step 2: Leaving the swarm...
docker swarm leave --force

echo Step 3: Building new Docker image 'kadlab:latest'...
docker build --no-cache -t kadlab:latest .

echo Step 4: Initializing new swarm...
docker swarm init

echo Step 5: Deploying stack 'kademlia-app'...

REM Load variables from the .env file into the current shell session
for /f "delims=" %%a in (.env) do (set "%%a")

REM Now the deploy command will have access to the variables
docker stack deploy -c docker-compose.yml kademlia-app

echo --- Deployment Complete ---