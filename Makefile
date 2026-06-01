MATRIX ?= images-matrix.yaml
PLAN ?= batch-plan.nvidia-cuda.practical-3apps.tsv
OUT_DIR ?= dist/argo
BIN ?= ./bin/matrix-ci
UPLOAD_DEST ?= myminio/kaniko-contexts/k8ace/context.tar.gz
GO_ENV ?= GOPROXY=https://goproxy.cn,direct GONOSUMDB=* GONOPROXY=

.PHONY: build test clean package upload render apply create dry-run base-render base-create practical-render practical-create help

build: ## 编译 matrix-ci
	$(GO_ENV) go build -buildvcs=false -o $(BIN) ./cmd/matrix-ci

test: ## 运行全部测试
	$(GO_ENV) go test ./... -count=1

clean: ## 清理本地产物
	rm -rf bin dist/context dist/argo dist/argo-* matrix-ci

package: ## 打包构建上下文
	bash ./hack/package_context.sh

upload: package ## 上传上下文到 MinIO
	mc cp ./dist/context/context.tar.gz $(UPLOAD_DEST)

render: build ## 按 PLAN 渲染 WorkflowTemplate
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR)

apply: build ## 按 PLAN 渲染并 apply WorkflowTemplate
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR) --apply

create: build ## 按 PLAN 完整创建 Workflow
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR) --create

dry-run: build ## 打印将执行的命令
	$(BIN) batch --plan $(PLAN) --matrix $(MATRIX) --out-dir $(OUT_DIR) --create --dry-run

base-render: build ## 渲染共享 base 的 WorkflowTemplate
	$(BIN) batch --plan ./batch-plan.nvidia-cuda.base12.4.tsv --matrix $(MATRIX) --out-dir ./dist/argo-base12.4

base-create: build ## 创建共享 base 的 Workflow
	$(BIN) batch --plan ./batch-plan.nvidia-cuda.base12.4.tsv --matrix $(MATRIX) --out-dir ./dist/argo-base12.4 --create

practical-render: build ## 渲染三个实战 app 的 WorkflowTemplate
	$(BIN) batch --plan ./batch-plan.nvidia-cuda.practical-3apps.tsv --matrix $(MATRIX) --out-dir ./dist/argo-practical-3apps

practical-create: build ## 创建三个实战 app 的 Workflow
	$(BIN) batch --plan ./batch-plan.nvidia-cuda.practical-3apps.tsv --matrix $(MATRIX) --out-dir ./dist/argo-practical-3apps --create

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \\033[36m%-18s\\033[0m %s\\n", $$1, $$2}'
