
# Hyper-V DNS

```C:\Windows\System32\drivers\etc\hosts.ics``` を開く

# IP アドレス固定

## Hyper-V

1. 「仮想スイッチマネージャー」から内部ネットワークの仮想スイッチを新規作成
1. コントロールパネルの「ネットワークと共有センター」から，作成した内部スイッチのプロパティを開き，IP アドレスを固定する
1. 各 VM の「設定」を開き，「ハードウェアの追加」タブで「ネットワークアダプタ」を選択し，作成した内部スイッチを追加する

- Default Switch と内部スイッチの両方を付けておくことで，ゲスト OS からインターネットに接続できる状態を維持しつつ内部ネットワークに接続できる．

## ゲスト OS

1. ```sudo vim /etc/netplan/01-network-manager-all.yaml```
```
# Let NetworkManager manage all devices on this system
network:
  version: 2
  renderer: NetworkManager
# ここから追記
  ethernets:
    eth1:
      addresses:
      - 192.168.0.11/24
      gateway4: 192.168.0.10
      dhcp4: false
      dhcp6: false
      accept-ra: false
      nameservers:
        addresses:
          - 192.168.0.10

```
2. ```sudo netplan apply```


# SSH 公開鍵認証

1. ```ssh-keygen -t rsa```
1. ```ssh-copy-id -i .ssh/id_rsa.pub <user>@192.168.0.12```
1. ```ssh <user>@192.168.0.12```

# ポートフォワーディング

## SSH ポートフォワーディング
```
ssh -L <A-port>:<B-addr>:<B-port> -N -g <A-addr>
```
- -N : シェルを起動しない
- -g : localhost 以外からのフォワーディングを許可

- 例）```192.168.0.12:18080``` => ```192.168.0.13:28080```：
```
ssh -L 18080:192.168.0.13:28080 -N -g 192.168.0.12
```

## kubectl ポートフォワーディング
```
sudo kubectl port-forward --address 0.0.0.0 <pod-name> <local-port>:<pod-port>
```
- 例）```0.0.0.0:28080``` => ```app-restore-pod:8080```：
```
sudo kubectl port-forward --address 0.0.0.0 app-restore-pod 28080:8080
```

# rsync

```
rsync -rlOtcv /tmp/testdir rsync://192.168.0.12:10873/tmp
```

# 疎通確認

1. サーバ B の Pod でリッスン \
  ``` [B.pod] # nc -l -s 0.0.0.0 -p 8080 ```
1. サーバ B のホストで kubectl port-forward \
  ```[B.local] $ sudo kubectl port-forward --address 0.0.0.0 app-restore-pod 28080:8080 ```
1. サーバ A のホストで ssh ポートフォワーディング \
  ```[A.local] $ ssh -L 18080:192.168.0.13:28080 -N -g 192.168.0.12```
1. サーバ A の Pod から送信 \
  ```[A.pod] # nc 192.168.0.12:18080```

# Minikube

- メモリを制限
```
minikube start --memory=2200mb
```

# カーネル ダウングレード

コンテナに対する CRIU は Linux Kernel バージョン 5.0.0-23-generic より後のバージョンで ```Error (criu/files-reg.c:1338): Can't lookup mount=XXX for fd=-3 path=XXX``` のようなエラーが出て使えないため．
詳細: https://github.com/checkpoint-restore/criu/issues/860

```
sudo apt install linux-image-5.0.0-23-generic linux-headers-5.0.0-23-generic linux-modules-extra-5.0.0-23-generic
sudo add-apt-repository ppa:danielrichter2007/grub-customizer
sudo apt-get install grub-customizer
# Grub customizer (GUI アプリ) を開いて設定
sudo vim /etc/apt/preferences.d/linux-kernel.pref
# ---
Package: linux-generic
Pin: version 5.0.0.23*
Pin-Priority: 1001

Package: linux-headers-generic
Pin: version 5.0.0.23*
Pin-Priority: 1001

Package: linux-image-generic
Pin: version 5.0.0.23*
Pin-Priority: 1001
# ---
```

# CRIU

## Checkpoint/Restore
```
# Checkpoint
criu dump --tree <pid> -D /tmp/path/to/images --tcp-established --shell-job
# Restore
unshare -p -m --fork --mount-proc criu restore -D /tmp/path/to/images --tcp-established --shell-job
```

# Docker プライベートレジストリ

ここでレジストリサーバのアドレスは 192.168.0.12 とする．

## サーバ (レジストリ)

1. レジストリのイメージをプル ```[Server] sudo docker pull registry```

1. 非 TLS 通信を許可 ```[Server] sudo vim /etc/docker/daemon.json```
```
{
        "insecure-registries":["192.168.0.12:5000"]
}
```

1. Docker 再起動 ```[Server] sudo systemctl restart docker```

1. パスワードファイル作成
```
[Server]
mkdir auth
htpasswd -Bbn admin password > auth/htpasswd
```

1. レジストリを起動
```
[Server]
sudo docker run -d -p 5000:5000 --name registry -v `pwd`/auth:/auth -e "REGISTRY_AUTH=htpasswd" -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" registry
```

1. レジストリにログイン (Username: admin, Password: password) ```docker login 192.168.0.12:5000``` 

1. タグ付け ```[Server] sudo docker build . -t 192.168.0.12:5000/app-sample```

1. アプリのイメージを Push ```[Server] sudo docker push 192.168.0.12:5000/app-sample```

## クライアント

1. 非 TLS 通信を許可 ```[Client] sudo vim /etc/docker/daemon.json```
```
{
        "insecure-registries":["192.168.0.12:5000"]
}
```

1. Docker 再起動 ```[Client] sudo systemctl restart docker```

1. レジストリにログイン (Username: admin, Password: password) ```docker login 192.168.0.12:5000```

1. イメージを Pull ```[Client] sudo docker pull 192.168.0.12:5000/app-sample``` 

### クライアントで Kubernetes が Pull する場合

1. レジストリにログイン

1. regcred を登録
```
sudo kubectl create secret generic regcred --from-file=.dockerconfigjson=/home/x0unnamed/.docker/config.json --type=kubernetes.io/dockerconfigjson
```
