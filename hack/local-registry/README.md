# K8Ace Local Registry

本目录用于启动服务器本地 Docker Registry，以及一个轻量 Web UI。

## 服务组成

- `registry`: 本地镜像仓库，端口 `5000`。
- `registry-ui`: 镜像仓库网页管理界面，端口 `8088`。

当前产线构建出来的镜像会推送到：

```text
172.20.47.182:5000/k8ace/...
```

## 启动与查看

```bash
cd /home/xuefeng/newimage/images/hack/local-registry
docker compose up -d
docker compose ps
```

查看 Registry API：

```bash
curl http://127.0.0.1:5000/v2/_catalog
```

查看 Web UI：

```text
http://172.20.47.182:8088
```

如果不想直接暴露给局域网，也可以用 SSH 转发：

```powershell
ssh -i "E:\rsa\id_rsa_2048" -L 8088:127.0.0.1:8088 xuefeng@172.20.47.182
```

然后本地浏览器打开：

```text
http://127.0.0.1:8088
```

## 拉取和运行镜像

```bash
docker pull 172.20.47.182:5000/k8ace/<repo>:<tag>
```

服务型镜像示例：

```bash
docker run --rm -it \
  --gpus all \
  -p 8188:8188 \
  172.20.47.182:5000/k8ace/comfyui0.22.0-nvidia-comfyui-service-cuda124-dev:latest
```

## 删除和整理

Web UI 可以删除 tag，但 Docker Registry 删除后不会立即释放磁盘空间。
真正释放空间需要执行 garbage collect。

建议流程：

```bash
cd /home/xuefeng/newimage/images/hack/local-registry

docker compose stop registry-ui
docker compose stop registry

docker compose run --rm registry registry garbage-collect /etc/docker/registry/config.yml

docker compose up -d
```

如果要彻底清空整个 Registry，慎用：

```bash
cd /home/xuefeng/newimage/images/hack/local-registry

docker compose down
docker volume rm local-registry_registry-data
docker compose up -d
```

## 回滚 Compose

添加 Web UI 前的 compose 文件会备份为：

```text
docker-compose.yaml.bak-registry-ui-YYYYMMDD-HHMMSS
```

需要回滚时复制回来，然后重启：

```bash
cp docker-compose.yaml.bak-registry-ui-YYYYMMDD-HHMMSS docker-compose.yaml
docker compose up -d
```
