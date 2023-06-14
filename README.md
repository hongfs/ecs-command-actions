# ECS Command Actions

作用于 GitHub Actions，向阿里云 ECS 服务器发送命令执行。比如服务器拉取最新代码、拉取最新容器镜像、重启容器等。

```yaml
- uses: docker://ghcr.io/hongfs/ecs-command-actions:main@sha256:9a9c0c497c4e856625660948f1bf21fb6912744901537b5bcc924e15d16ce6c5
  env:
    ALIYUN_ACCESS_KEY_ID: ${{ secrets.ALIYUN_ACCESS_KEY_ID }}
    ALIYUN_ACCESS_KEY_SECRET: ${{ secrets.ALIYUN_ACCESS_KEY_SECRET }}
    ALIYUN_REGION: cn-shenzhen
    # 配置多个标签来进行过滤
    ALIYUN_TAGS: tags=1;tags2=2
    # 执行的命令
    ALIYUN_SCRIPT: >-
      #!/bin/bash
      echo "hello world"
```