#!/bin/bash
# ============================================================================
# 项目打包工具 - 一键执行脚本
# ============================================================================
# 
# 功能：
#   1. 自动创建虚拟环境 (.venv)
#   2. 安装所需依赖
#   3. 执行打包脚本，生成 markdown、html 和截图
#
# 使用方法：
#   ./run.sh                              # 打包 soulmark 目录
#   ./run.sh -i ../myproject              # 打包指定目录
#   ./run.sh -i ../myproject -o results   # 打包并输出到指定目录
#
# ============================================================================

# ============================================================================
# 默认配置
# ============================================================================

# 默认输入目录：soulmark 项目根目录（上一级目录）
DEFAULT_INPUT_DIR=".."

# 默认输出目录
DEFAULT_OUTPUT_DIR="output"

# 虚拟环境目录名称
VENV_DIR=".venv"

# ============================================================================
# 解析命令行参数
# ============================================================================

INPUT_DIR=""
OUTPUT_DIR=""
SCREENSHOT_MODE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--input)
            INPUT_DIR="$2"
            shift
            shift
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift
            shift
            ;;
        -m|--screenshot-mode)
            SCREENSHOT_MODE="$2"
            shift
            shift
            ;;
        -h|--help)
            echo "使用方法: ./run.sh [选项]"
            echo ""
            echo "选项:"
            echo "  -i, --input <目录>          输入目录路径"
            echo "  -o, --output <目录>         输出目录路径"
            echo "  -m, --screenshot-mode <模式> 截图模式: auto(默认), full, segment, both"
            echo "  -h, --help                  显示帮助信息"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            shift
            ;;
    esac
done

if [ -z "$INPUT_DIR" ]; then
    INPUT_DIR="$DEFAULT_INPUT_DIR"
fi

if [ -z "$OUTPUT_DIR" ]; then
    OUTPUT_DIR="$DEFAULT_OUTPUT_DIR"
fi

# ============================================================================
# 主流程
# ============================================================================

echo "============================================================"
echo "📦 项目打包工具 - 一键执行脚本"
echo "============================================================"
echo ""

# 步骤 1: 检查虚拟环境是否存在
echo "🔍 步骤 1/5: 检查虚拟环境..."
if [ -d "$VENV_DIR" ]; then
    echo "   ✅ 虚拟环境已存在: $VENV_DIR"
else
    echo "   📁 创建虚拟环境: $VENV_DIR"
    python3 -m venv "$VENV_DIR"
    if [ $? -eq 0 ]; then
        echo "   ✅ 虚拟环境创建成功"
    else
        echo "   ❌ 虚拟环境创建失败"
        exit 1
    fi
fi

# 步骤 2: 激活虚拟环境
echo ""
echo "🔌 步骤 2/5: 激活虚拟环境..."
source "$VENV_DIR/bin/activate"
if [ $? -eq 0 ]; then
    echo "   ✅ 虚拟环境已激活"
else
    echo "   ❌ 虚拟环境激活失败"
    exit 1
fi

# 步骤 3: 安装依赖
echo ""
echo "📥 步骤 3/5: 安装依赖..."
pip install -q -i https://pypi.tuna.tsinghua.edu.cn/simple -r requirements.txt
if [ $? -eq 0 ]; then
    echo "   ✅ 依赖安装成功"
else
    echo "   ❌ 依赖安装失败"
    exit 1
fi

# 步骤 4: 安装 Playwright 浏览器
echo ""
echo "🌐 步骤 4/5: 安装 Playwright 浏览器..."
python -m playwright install chromium
if [ $? -eq 0 ]; then
    echo "   ✅ 浏览器安装成功"
else
    echo "   ⚠️  浏览器安装失败，截图功能可能无法使用"
fi

# 步骤 5: 执行打包脚本
echo ""
echo "🚀 步骤 5/5: 执行打包脚本..."
echo "   📂 输入目录: $INPUT_DIR"
echo "   📂 输出目录: $OUTPUT_DIR"

SCREENSHOT_ARGS=""
if [ -n "$SCREENSHOT_MODE" ]; then
    SCREENSHOT_ARGS="--screenshot-mode $SCREENSHOT_MODE"
    echo "   📸 截图模式: $SCREENSHOT_MODE"
fi
echo ""

python main.py --input "$INPUT_DIR" --output "$OUTPUT_DIR" $SCREENSHOT_ARGS

# 检查执行结果
if [ $? -eq 0 ]; then
    echo ""
    echo "============================================================"
    echo "✨ 打包完成！"
    echo "============================================================"
else
    echo ""
    echo "============================================================"
    echo "❌ 打包失败，请检查错误信息"
    echo "============================================================"
    exit 1
fi
