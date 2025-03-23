# SETMaster
Репозиторій створено для розміщення прототипу рішення для дипломної роботи 

Репозиторій складається з 3-х розділів
* config-files - конфігураційні файли для проекту
* hubble-plugin - плагін для зв'язку моніторингових систем hubble та falco
* module-engine - модуль взаємодії з зовнішніми системами

Розгортання повного проекту дослідницької системи складається з кількох кроків. Всі кроки виконуються послідовно в операційній системі Linux.
## Розгортання проекту
### 0. Завантаження репозиторію
```bash
git clone https://github.com/vzinenko-set/SETMaster.git
```
### 1. Встановлення k3s
```
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC='--write-kubeconfig-mode=644 --disable-network-policy' sh -
mkdir .kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chmod 600 ~/.kube/config
sudo chown user_name:user_name ./.kube/config
export KUBECONFIG=~/.kube/config
```
### 2. Встановлення helm (менеджер пакетів)
```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```
### 3. Встановлення vCluster
```bash
kubectl create namespace virtual-cluster
curl -L -o vcluster "https://github.com/loft-sh/vcluster/releases/latest/download/vcluster-linux-amd64" && sudo install -c -m 0755 vcluster /usr/local/bin
vcluster create honeypot --namespace virtual-cluster --values config-files/vcluster.yaml
```
### 4. Встановлення cilium + hubble
```
#---=== install cilium cli ===---
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
CLI_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then CLI_ARCH=arm64; fi
curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
sha256sum --check cilium-linux-${CLI_ARCH}.tar.gz.sha256sum
sudo tar xzvfC cilium-linux-${CLI_ARCH}.tar.gz /usr/local/bin
rm cilium-linux-${CLI_ARCH}.tar.gz{,.sha256sum}
#---=== finish cilium cli ===---

cilium install --version 1.17.2 --set=ipam.operator.clusterPoolIPv4PodCIDRList="10.42.0.0/16"
cilium status --wait
cilium hubble enable

#---=== install hubble cli ===---
HUBBLE_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/hubble/master/stable.txt)
HUBBLE_ARCH=amd64
if [ "$(uname -m)" = "aarch64" ]; then HUBBLE_ARCH=arm64; fi
curl -L --fail --remote-name-all https://github.com/cilium/hubble/releases/download/$HUBBLE_VERSION/hubble-linux-${HUBBLE_ARCH}.tar.gz{,.sha256sum}
sha256sum --check hubble-linux-${HUBBLE_ARCH}.tar.gz.sha256sum
sudo tar xzvfC hubble-linux-${HUBBLE_ARCH}.tar.gz /usr/local/bin
rm hubble-linux-${HUBBLE_ARCH}.tar.gz{,.sha256sum}
#---=== finish hubble cli ===---

cilium hubble port-forward&
```
### 5. Встановлення falco + falcosidekick
```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update
kubectl create namespace falco
helm install falco -n falco --set tty=true falcosecurity/falco \
  --set falcosidekick.enabled=true \
  --set falcosidekick.webui.enabled=true \
  --set collectors.kubernetes.enabled=true
  -f config-files/falco-values.yaml
```
### 6. Збірка плагіну для falco
```bash
cd hubble-plugin
go build -buildmode=c-shared -o hubble.so *.go
kubectl cp hubble.so falco-pod:/usr/share/falco/plugins/hubble.so -n falco
```
### 7. Налаштування slack для взаємоді (за потреби)
```
Залогінитися на сторінці https://api.slack.com
Створити додадток App.
Додати callback url в налаштуваннях.
Взяти url для webhook, токен для бота та ID-каналу в котрий будуть надсилатися повідомлення.
Додати бота до каналу.
Отримані значення внести до конфігураційного файлу модуля - config.yaml
```
### 8. Додавання honeypot в кластер
```bash
cd config-files
dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
dnf install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
docker image build .
docker image push reponame
vcluster connect honeypot
kubectl appy -f honeypot-k3s-config.yaml
```
### 9. Модуль взаємодії з зовнішніми системами (module-engine)
```bash
cd module-engine
#Edit config file
go build -o moduleEngine cmd/server/main.go
./moduleEngine
```