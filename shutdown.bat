@echo off

echo Step 1: Removing old stack 'kademlia-app'...
docker stack rm kademlia-app

echo Step 2: Leaving the swarm...
docker swarm leave --force