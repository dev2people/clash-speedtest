#!/bin/bash

# 定义镜像名称
IMAGE_NAME="dev2people/clashsub-tools-clashspeedtest-v0"

# 获取当前日期和时间作为标签（例如：20250623-073151）
# 确保在运行脚本的机器上，日期时间格式与此处一致
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")

echo "---"
echo "正在移除旧的本地镜像: ${IMAGE_NAME}:latest"
docker rmi "${IMAGE_NAME}:latest"

echo "---"
echo "正在构建 Docker 镜像，同时打上 ${TIMESTAMP} 和 latest 标签"
# 构建镜像并打上两个标签
docker build -f ./build.dockerfile -t "${IMAGE_NAME}:${TIMESTAMP}" -t "${IMAGE_NAME}:latest" .

echo "---"
echo "正在推送 Docker 镜像: ${IMAGE_NAME}:${TIMESTAMP}"
docker push "${IMAGE_NAME}:${TIMESTAMP}"

echo "---"
echo "正在推送 Docker 镜像: ${IMAGE_NAME}:latest"
docker push "${IMAGE_NAME}:latest"

echo "---"
echo "Docker 镜像发布完成！"