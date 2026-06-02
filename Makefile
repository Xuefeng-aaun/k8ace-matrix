# ==============================================================================
# K8Ace Matrix Makefile
# ==============================================================================
#
# 推荐使用方式：
#   1. 先写 batch-plan.tsv，说明“我要生产哪些 base/app”。
#   2. 先运行 make preview PLAN=你的plan.tsv，确认会创建哪些 Argo Workflow。
#   3. 确认无误后运行 make produce PLAN=你的plan.tsv，正式提交产线。
#
# preview 和 produce 的区别：
#   - preview：只预演，不上传 context，不创建真实 Workflow，不生产镜像。
#   - produce：正式执行，会打包 context、上传 MinIO、创建 Workflow，并等待结果。
#
# OUT_DIR 不需要手写，会自动使用 dist/<plan文件名去掉.tsv>。
# 例如 PLAN=batch-plan.nvidia-cuda.practical-3apps.tsv 时，
# OUT_DIR 默认为 dist/batch-plan.nvidia-cuda.practical-3apps。
#
# ==============================================================================

# 矩阵配置文件。这里记录 base/app/Argo/Kaniko/MinIO/Registry 等产线配置。
MATRIX ?= images-matrix.yaml

# 批量生产计划。使用者最常改这个变量。
PLAN ?= batch-plan.nvidia-cuda.practical-3apps.tsv

# 根据 PLAN 自动生成输出目录名。
PLAN_NAME = $(basename $(notdir $(PLAN)))

# 生成出来的 WorkflowTemplate/Workflow YAML 存放目录，默认不需要手写。
OUT_DIR ?= dist/$(PLAN_NAME)

# 编译后的 matrix-ci 程序路径。
BIN ?= ./bin/matrix-ci

# context.tar.gz 上传到 MinIO 的目标位置。
# 这个地址需要和 images-matrix.yaml 里的 ci_cd.argo_workflows.build_context.default 对应。
UPLOAD_DEST ?= myminio/kaniko-contexts/k8ace/context.tar.gz

# Go 编译/测试时使用的代理配置。弱网环境下更稳定。
GO_ENV ?= GOPROXY=https://goproxy.cn,direct GONOSUMDB=* GONOPROXY=

DEMO_PLAN := batch-plan.nvidia-cuda.practical-3apps.tsv

.PHONY: help build test clean package upload preview render apply create produce demo demo-preview base-preview base-produce

help: ## 显示常用命令
	@echo "K8Ace Matrix 使用入口"
	@echo ""
	@echo "你通常只需要做两件事："
	@echo "  1. 写好 batch-plan.tsv"
	@echo "  2. 先 preview，确认无误后 produce"
	@echo ""
	@echo "最常用命令："
	@echo "  make preview PLAN=你的plan.tsv"
	@echo "  make produce PLAN=你的plan.tsv"
	@echo "  输出目录会自动使用 dist/<plan文件名>"
	@echo ""
	@echo "preview：只预演，不上传 MinIO，不创建真实 Workflow，不生产镜像"
	@echo "produce：正式生产，会打包 context、上传 MinIO、创建 Workflow，并等待结果"
	@echo ""
	@echo "当前三应用 demo："
	@echo "  make demo-preview"
	@echo "  make demo"
	@echo ""
	@echo "可配置变量："
	@echo "  MATRIX=$(MATRIX)"
	@echo "  PLAN=$(PLAN)"
	@echo "  OUT_DIR=$(OUT_DIR)"
	@echo "  UPLOAD_DEST=$(UPLOAD_DEST)"

build: ## 编译 matrix-ci
	$(GO_ENV) go build -buildvcs=false -o $(BIN) ./cmd/matrix-ci

test: ## 运行 Go 单元测试
	$(GO_ENV) go test ./... -count=1

clean: ## 清理本地生成物
	rm -rf bin dist/context dist/argo dist/argo-* dist/batch-plan* matrix-ci

package: ## 打包构建上下文 context.tar.gz
	bash ./hack/package_context.sh

upload: package ## 上传构建上下文到 MinIO
	mc cp ./dist/context/context.tar.gz $(UPLOAD_DEST)

preview: build ## 预演 PLAN：只生成 YAML 并打印 kubectl 命令，不真正提交
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR) --create --dry-run

render: build ## 按 PLAN 只渲染 WorkflowTemplate/Workflow 文件
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR)

apply: build ## 只提交 WorkflowTemplate，不创建 Workflow，通常调试时才用
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR) --apply

create: build ## 创建真实 Workflow，并等待 Argo 返回阶段结果；不会自动打包/上传 context
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR) --create

produce: build test upload create ## 正式生产：编译、测试、打包、上传、创建 Workflow 并等待反馈

demo-preview: PLAN=$(DEMO_PLAN)
demo-preview: preview ## 预演当前三应用 demo

demo: PLAN=$(DEMO_PLAN)
demo: produce ## 生产当前三应用 demo

base-preview: PLAN=batch-plan.nvidia-cuda.base12.4.tsv
base-preview: preview ## 预演共享 base image 生产

base-produce: PLAN=batch-plan.nvidia-cuda.base12.4.tsv
base-produce: produce ## 生产共享 base image
