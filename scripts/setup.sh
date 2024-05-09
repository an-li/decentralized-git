sudo apt-get update
sudo apt-get install git curl docker-compose -y

# Make sure the Docker daemon is running.
sudo systemctl start docker

# Add your user to the Docker group.
sudo usermod -a -G docker $USER

# Check version numbers  
docker --version
docker-compose --version

# Optional: If you want the Docker daemon to start when the system starts, use the following:
sudo systemctl enable docker

# Install JQ
sudo apt-get install jq -y

# Install Go
curl -ssLO https://go.dev/dl/go1.22.1.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.1.linux-amd64.tar.gz

# Install Docker and binaries for Hyperledger Fabric
curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh && chmod +x install-fabric.sh
./install-fabric.sh binary
rm -rf install-fabric.sh

echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install ipfs
wget https://dist.ipfs.io/go-ipfs/v0.7.0/go-ipfs_v0.7.0_linux-amd64.tar.gz
tar -xvzf go-ipfs_v0.7.0_linux-amd64.tar.gz
cd go-ipfs
sudo bash install.sh
cd ..
rm -rf go-ipfs
rm -rf go-ipfs_v0.7.0_linux-amd64.tar.gz

# Initialize local IPFS client
rm -rf ~/.ipfs
ipfs init

# Install requirements for Python client
sudo pip install -r ../client/requirements.txt --no-cache-dir
git clone https://github.com/hyperledger/fabric-sdk-py.git
cd fabric-sdk-py
sudo python3 setup.py
cd ..
rm -rf fabric-sdk-py