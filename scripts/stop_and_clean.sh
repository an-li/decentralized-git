# Stop fabric network
cd ~/fabric-samples/test-network
./network.sh down
docker system prune -fa

# Clean up ipfs
cd ~
rm -rf .ipfs