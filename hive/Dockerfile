# ============================================================
#  Hive 蜂巢 · 多阶段构建 Dockerfile（构建上下文 = 本 hive/ 目录）
#  阶段1 用 Maven 编译出可执行 jar，阶段2 只装 JRE 运行，镜像更小更安全
#  由 .github/workflows/docker-publish.yml 自动构建并推送到 GHCR
# ============================================================

# ---- 阶段 1：构建 ----
FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /build

# 先单独拷 pom 预拉依赖，利用 Docker 层缓存：源码改动不会触发重新下载依赖
COPY pom.xml .
RUN mvn -B -q dependency:go-offline

# 再拷源码并打包（跳过测试编译与执行，测试需连库，部署构建阶段不便执行）
COPY src ./src
RUN mvn -B -q -Dmaven.test.skip=true clean package

# ---- 阶段 2：运行 ----
FROM eclipse-temurin:21-jre
WORKDIR /app

# 以非 root 用户运行，降低风险
RUN useradd -r -u 1001 -d /app hive

# 拷入构建好的 jar（finalName=hive → target/hive.jar）
COPY --from=build /build/target/hive.jar app.jar

# 上传目录（compose 会把它挂成数据卷以持久化图片）
RUN mkdir -p /app/uploads && chown -R hive:hive /app
USER hive

# 容器内固定监听 8080；对外端口由 compose 的 ports 映射决定
ENV SERVER_PORT=8080 \
    HIVE_UPLOAD_DIR=/app/uploads \
    TZ=Asia/Shanghai
EXPOSE 8080

ENTRYPOINT ["java", "-jar", "/app/app.jar"]
