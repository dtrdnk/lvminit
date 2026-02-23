

#### Install colima instead docker-desktop
```shell
brew install colima lima-additional-guestagents
```

#### Necessary arch option for correct works with loop devices
```shell
colima start --arch x86_64
```

#### Go to colima shell
```shell
colima ssh
```

#### Update repos
```shell
sudo apt-get update
```

#### Install necessary dependencies for e2e
```shell
sudo apt-get install -y util-linux make golang-go vim --no-install-recommends
```

#### Install Kind for AMD64 x86_64 architecture
```shell
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.31.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

#### Install helm
```shell
wget https://get.helm.sh/helm-v3.20.0-linux-amd64.tar.gz && \
  tar -xvf helm-v3.20.0-linux-amd64.tar.gz && \
  sudo mv linux-amd64/helm /usr/local/bin
```

#### Installl kubectl
```shell
curl -LO "https://dl.k8s.io/release/$(curl -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/
```

#### Run e2e
```shell
make e2e
```
