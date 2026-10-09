.PHONY: build

# 前端与内嵌前端的服务端只有一个编译入口，本地打包与 CI 共用 tools/gotocc_build.py。
build:
	@python3 tools/gotocc_build.py --output backend/bin/sub2api
