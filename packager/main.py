#!/usr/bin/env python3
"""
项目打包工具
============

功能：
1. 自动汇总指定目录下的所有文件
2. 生成目录树总览
3. 统计各目录下的文件数
4. 按顺序整理文件内容到 markdown
5. 生成 html 和截图

使用方法：
    python main.py --input <输入目录> --output <输出目录>
"""

import os
import argparse
from collections import defaultdict
from datetime import datetime

import markdown
from playwright.sync_api import sync_playwright


# ============================================================================
# 配置常量
# ============================================================================

# 需要排除的目录名称
EXCLUDE_DIRS = {
    'venv', '.venv', '__pycache__', '.git', '.idea', 'node_modules',
    '.DS_Store', 'dist', 'build', '.eggs', 'output', 'packager'
}

# 需要排除的文件模式
EXCLUDE_FILES = {
    '.DS_Store', 'Thumbs.db', '.gitignore', '.env'
}

# 大文件阈值（10MB）
MAX_FILE_SIZE = 10 * 1024 * 1024

# 文件扩展名到语言的映射
LANGUAGE_MAP = {
    '.py': 'python',
    '.js': 'javascript',
    '.ts': 'typescript',
    '.html': 'html',
    '.css': 'css',
    '.json': 'json',
    '.yaml': 'yaml',
    '.yml': 'yaml',
    '.md': 'markdown',
    '.txt': 'text',
    '.sh': 'bash',
    '.sql': 'sql',
    '.xml': 'xml',
    '.java': 'java',
    '.go': 'go',
    '.rs': 'rust',
    '.cpp': 'cpp',
    '.c': 'c',
    '.h': 'c',
    '.hpp': 'cpp',
}


# ============================================================================
# 文件收集和过滤功能
# ============================================================================

def should_exclude_dir(dir_name: str) -> bool:
    """
    判断目录是否应该被排除
    
    Args:
        dir_name: 目录名称
        
    Returns:
        bool: 如果应该排除返回 True，否则返回 False
    """
    # 检查是否在排除列表中
    if dir_name in EXCLUDE_DIRS:
        return True
    # 检查是否以点开头（隐藏目录）
    if dir_name.startswith('.'):
        return True
    return False


def should_exclude_file(file_name: str) -> bool:
    """
    判断文件是否应该被排除
    
    Args:
        file_name: 文件名称
        
    Returns:
        bool: 如果应该排除返回 True，否则返回 False
    """
    # 检查是否在排除列表中
    if file_name in EXCLUDE_FILES:
        return True
    # 检查是否以点开头（隐藏文件）
    if file_name.startswith('.'):
        return True
    return False


def collect_files_and_generate_tree(input_dir: str) -> tuple:
    """
    收集目录下的所有文件并生成目录树（一次遍历完成）
    
    Args:
        input_dir: 输入目录路径
        
    Returns:
        tuple: (files, tree_lines, dir_counts)
            - files: 包含 (相对路径, 绝对路径) 元组的列表
            - tree_lines: 目录树的字符串列表
            - dir_counts: 目录路径到文件数量的映射
    """
    files = []
    tree_lines = []
    dir_counts = defaultdict(int)
    
    for root, dirs, filenames in os.walk(input_dir):
        # 过滤掉需要排除的目录（修改 dirs 会影响 os.walk 的遍历）
        dirs[:] = [d for d in dirs if not should_exclude_dir(d)]
        
        # 计算缩进级别
        level = root.replace(input_dir, '').count(os.sep)
        indent = '  ' * level
        
        # 添加目录名称
        dir_name = os.path.basename(root) if root != input_dir else os.path.basename(input_dir)
        tree_lines.append(f"{indent}📁 {dir_name}/")
        
        # 统计当前目录的文件数
        current_dir_files = 0
        
        # 添加文件名称（过滤掉需要排除的文件）
        subindent = '  ' * (level + 1)
        for filename in sorted(filenames):
            if not should_exclude_file(filename):
                # 根据文件扩展名选择图标
                ext = os.path.splitext(filename)[1].lower()
                if ext in ['.py']:
                    icon = '🐍'
                elif ext in ['.js', '.ts']:
                    icon = '📜'
                elif ext in ['.html', '.css']:
                    icon = '🌐'
                elif ext in ['.md']:
                    icon = '📝'
                elif ext in ['.json', '.yaml', '.yml']:
                    icon = '⚙️'
                else:
                    icon = '📄'
                tree_lines.append(f"{subindent}{icon} {filename}")
                
                # 收集文件
                file_path = os.path.join(root, filename)
                relative_path = os.path.relpath(file_path, input_dir)
                files.append((relative_path, file_path))
                
                current_dir_files += 1
        
        # 统计目录文件数
        if current_dir_files > 0:
            dir_path = os.path.relpath(root, input_dir)
            if not dir_path or dir_path == '.':
                dir_counts['[根目录]'] = current_dir_files
            else:
                dir_counts[dir_path] = current_dir_files
    
    # 按相对路径排序
    files.sort(key=lambda x: x[0])
    
    return files, tree_lines, dict(sorted(dir_counts.items()))


def generate_directory_tree(tree_lines: list) -> str:
    """
    将目录树列表转换为字符串
    
    Args:
        tree_lines: 目录树的字符串列表
        
    Returns:
        str: 目录树的字符串表示
    """
    return '\n'.join(tree_lines)


# ============================================================================
# Markdown 生成功能
# ============================================================================

def get_file_language(file_path: str) -> str:
    """
    根据文件扩展名获取语言标识
    
    Args:
        file_path: 文件路径
        
    Returns:
        str: 语言标识
    """
    ext = os.path.splitext(file_path)[1].lower()
    return LANGUAGE_MAP.get(ext, '')


def read_file_content(file_path: str) -> tuple:
    """
    读取文件内容
    
    Args:
        file_path: 文件路径
        
    Returns:
        tuple: (是否成功, 内容或错误信息)
    """
    try:
        # 检查文件大小
        file_size = os.path.getsize(file_path)
        if file_size > MAX_FILE_SIZE:
            return False, f"文件过大（{file_size / 1024 / 1024:.2f}MB），超过限制（{MAX_FILE_SIZE / 1024 / 1024:.2f}MB）"
        
        with open(file_path, 'r', encoding='utf-8') as f:
            return True, f.read()
    except UnicodeDecodeError:
        # 尝试其他编码
        try:
            with open(file_path, 'r', encoding='gbk') as f:
                return True, f.read()
        except:
            return False, "无法解码文件（二进制文件或不支持的编码）"
    except Exception as e:
        return False, f"读取文件失败: {str(e)}"


def generate_markdown(files: list, input_dir: str, tree_lines: list, dir_counts: dict) -> str:
    """
    生成完整的 markdown 内容
    
    Args:
        files: 文件列表
        input_dir: 输入目录路径
        tree_lines: 目录树的字符串列表
        dir_counts: 目录路径到文件数量的映射
        
    Returns:
        str: markdown 内容
    """
    md_content = []
    
    # ========================================================================
    # 标题和基本信息
    # ========================================================================
    md_content.append("# 📦 项目打包内容\n")
    md_content.append(f"**生成时间**: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
    md_content.append(f"**源目录**: `{os.path.abspath(input_dir)}`\n")
    md_content.append(f"**文件总数**: {len(files)}\n")
    
    # ========================================================================
    # 目录树总览
    # ========================================================================
    md_content.append("\n---\n")
    md_content.append("## 📂 目录树总览\n")
    md_content.append("```\n")
    md_content.append(generate_directory_tree(tree_lines))
    md_content.append("\n```\n")
    
    # ========================================================================
    # 各目录文件数统计
    # ========================================================================
    md_content.append("\n---\n")
    md_content.append("## 📊 各目录文件数统计\n")
    
    md_content.append("\n| 目录路径 | 文件数 |")
    md_content.append("\n|----------|--------|")
    for dir_path, count in dir_counts.items():
        md_content.append(f"\n| `{dir_path}` | {count} |")
    md_content.append("\n")
    
    # ========================================================================
    # 文件内容详情
    # ========================================================================
    md_content.append("\n---\n")
    md_content.append("## 📄 文件内容详情\n")
    
    # 按目录分组
    files_by_dir = defaultdict(list)
    for relative_path, file_path in files:
        dir_path = os.path.dirname(relative_path)
        if not dir_path:
            dir_path = '[根目录]'
        files_by_dir[dir_path].append((relative_path, file_path))
    
    # 计算总文件数
    total_files = len(files)
    processed_files = 0
    
    # 按目录顺序输出文件内容
    for dir_path in sorted(files_by_dir.keys()):
        # 添加目录标题
        md_content.append(f"\n### 📁 {dir_path}\n")
        
        for relative_path, file_path in files_by_dir[dir_path]:
            # 添加文件标题
            file_name = os.path.basename(file_path)
            md_content.append(f"\n#### `{file_name}`\n")
            md_content.append(f"> 路径: `{relative_path}`\n")
            
            # 读取并添加文件内容
            success, content = read_file_content(file_path)
            if success:
                language = get_file_language(file_path)
                md_content.append(f"\n```{language}\n")
                md_content.append(content)
                md_content.append("\n```\n")
            else:
                md_content.append(f"\n> ⚠️ {content}\n")
            
            # 更新进度
            processed_files += 1
            if processed_files % 5 == 0 or processed_files == total_files:
                print(f"   📄 处理文件 {processed_files}/{total_files}...")
    
    return ''.join(md_content)


# ============================================================================
# HTML 转换功能
# ============================================================================

def convert_to_html(md_content: str) -> str:
    """
    将 markdown 内容转换为带样式的 HTML
    
    Args:
        md_content: markdown 内容
        
    Returns:
        str: HTML 内容
    """
    # 使用 markdown 库转换
    html_body = markdown.markdown(
        md_content,
        extensions=[
            'fenced_code',      # 支持围栏代码块
            'codehilite',       # 代码高亮
            'tables',           # 支持表格
            'toc',              # 支持目录
        ]
    )
    
    # 构建完整的 HTML 文档
    full_html = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>项目打包内容</title>
    <style>
        /* 基础样式 */
        * {{
            box-sizing: border-box;
        }}
        
        body {{
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.8;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 40px 20px;
            background-color: #f8f9fa;
        }}
        
        /* 标题样式 */
        h1 {{
            color: #2c3e50;
            border-bottom: 3px solid #3498db;
            padding-bottom: 15px;
            margin-top: 0;
        }}
        
        h2 {{
            color: #34495e;
            border-bottom: 2px solid #bdc3c7;
            padding-bottom: 10px;
            margin-top: 40px;
        }}
        
        h3 {{
            color: #7f8c8d;
            margin-top: 30px;
        }}
        
        h4 {{
            color: #95a5a6;
            margin-top: 20px;
        }}
        
        /* 代码样式 */
        code {{
            font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
            background-color: #e8e8e8;
            padding: 2px 6px;
            border-radius: 4px;
            font-size: 0.9em;
        }}
        
        pre {{
            background-color: #2d2d2d;
            color: #f8f8f2;
            padding: 20px;
            border-radius: 8px;
            overflow-x: auto;
            margin: 15px 0;
        }}
        
        pre code {{
            background-color: transparent;
            padding: 0;
            color: inherit;
        }}
        
        /* 表格样式 */
        table {{
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
            background-color: white;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }}
        
        th, td {{
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #e1e1e1;
        }}
        
        th {{
            background-color: #3498db;
            color: white;
            font-weight: 600;
        }}
        
        tr:hover {{
            background-color: #f5f5f5;
        }}
        
        /* 引用样式 */
        blockquote {{
            border-left: 4px solid #3498db;
            margin: 20px 0;
            padding: 10px 20px;
            background-color: #ecf0f1;
            color: #7f8c8d;
        }}
        
        /* 分隔线样式 */
        hr {{
            border: none;
            height: 2px;
            background: linear-gradient(to right, transparent, #bdc3c7, transparent);
            margin: 40px 0;
        }}
        
        /* 列表样式 */
        ul, ol {{
            padding-left: 2em;
        }}
        
        li {{
            margin-bottom: 8px;
        }}
        
        /* 响应式设计 */
        @media (max-width: 768px) {{
            body {{
                padding: 20px 10px;
            }}
            
            pre {{
                padding: 15px;
            }}
        }}
    </style>
</head>
<body>
{html_body}
</body>
</html>"""
    
    return full_html


# ============================================================================
# 截图生成功能
# ============================================================================

def screenshot_full_page(page, output_path: str):
    """
    截取完整页面（长截图）
    
    Args:
        page: Playwright 页面对象
        output_path: 输出图片路径
    """
    page.screenshot(path=output_path, full_page=True)
    print(f"   ✅ 完整截图已保存: {output_path}")


def screenshot_segments(page, output_path: str) -> str:
    """
    分段截图，每段一个视口高度
    
    Args:
        page: Playwright 页面对象
        output_path: 输出图片路径（用于推导 segments 目录）
        
    Returns:
        str: 分段截图目录路径
    """
    total_height = page.evaluate('() => document.body.scrollHeight')
    viewport_height = page.viewport_size['height'] if page.viewport_size else 1080

    print(f"   ℹ️ 页面长度 {total_height}px，开始分段截图...")

    base_dir = os.path.dirname(output_path)
    segment_dir = os.path.join(base_dir, 'segments')
    os.makedirs(segment_dir, exist_ok=True)

    segments = []
    current_position = 0
    segment_index = 0

    while current_position < total_height:
        page.evaluate(f'window.scrollTo(0, {current_position})')
        page.wait_for_timeout(500)

        segment_path = os.path.join(segment_dir, f'segment_{segment_index}.png')
        page.screenshot(path=segment_path)
        segments.append(segment_path)

        current_position += viewport_height
        segment_index += 1

    segments_info = os.path.join(segment_dir, 'segments.txt')
    with open(segments_info, 'w') as f:
        f.write('\n'.join(segments))

    print(f"   ✅ 分段截图完成，共 {len(segments)} 段")
    print(f"   📁 分段截图保存在: {segment_dir}")

    return segment_dir


def generate_screenshot(html_content: str, output_path: str, mode: str = 'auto') -> list:
    """
    使用 Playwright 生成 HTML 页面的截图
    
    Args:
        html_content: HTML 内容
        output_path: 输出图片路径
        mode: 截图模式
            - 'auto':   页面 ≤3 视口高则截长图，否则分段
            - 'full':   强制截取完整长截图
            - 'segment': 强制分段截图
            - 'both':   同时生成长截图和分段截图
        
    Returns:
        list: 生成的截图路径列表
    """
    result_paths = []
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True)
            page = browser.new_page()
            page.set_content(html_content)
            page.wait_for_load_state('networkidle')

            total_height = page.evaluate('() => document.body.scrollHeight')
            viewport_height = page.viewport_size['height'] if page.viewport_size else 1080
            is_long = total_height > viewport_height * 3

            if mode == 'auto':
                if is_long:
                    screenshot_segments(page, output_path)
                else:
                    screenshot_full_page(page, output_path)

            elif mode == 'full':
                screenshot_full_page(page, output_path)
                result_paths.append(output_path)

            elif mode == 'segment':
                screenshot_segments(page, output_path)

            elif mode == 'both':
                screenshot_full_page(page, output_path)
                result_paths.append(output_path)
                screenshot_segments(page, output_path)

            browser.close()

        return result_paths

    except Exception as e:
        print(f"⚠️ 生成截图失败: {str(e)}")
        print("   提示: 请确保已安装 Playwright 浏览器，运行: playwright install chromium")
        return result_paths


# ============================================================================
# 主程序
# ============================================================================

def main():
    """
    主函数：解析参数并执行打包流程
    """
    # 解析命令行参数
    parser = argparse.ArgumentParser(
        description='📦 项目打包工具 - 将目录内容打包为 Markdown、HTML 和截图',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
    python main.py                                          # 打包当前目录
    python main.py -i ../myproject                          # 打包指定目录
    python main.py -i ../myproject -o results               # 指定输出目录
    python main.py -m both                                  # 同时生成完整截图和分段截图
    python main.py -m full                                  # 只生成完整长截图
    python main.py -m segment                               # 只生成分段截图
        """
    )
    
    parser.add_argument(
        '--input', '-i',
        default='.',
        help='输入目录路径（默认为当前目录）'
    )
    
    parser.add_argument(
        '--output', '-o',
        default='output',
        help='输出目录路径（默认为 output）'
    )
    
    parser.add_argument(
        '--screenshot-mode', '-m',
        default='auto',
        choices=['auto', 'full', 'segment', 'both'],
        help='截图模式: auto=自动选择(默认), full=完整长图, segment=分段截图, both=同时生成'
    )
    
    args = parser.parse_args()
    
    # 获取绝对路径
    input_dir = os.path.abspath(args.input)
    output_dir = os.path.abspath(args.output)
    
    # 验证输入目录
    if not os.path.isdir(input_dir):
        print(f"❌ 错误: 输入目录不存在: {input_dir}")
        return
    
    # 创建输出目录
    os.makedirs(output_dir, exist_ok=True)
    
    print("=" * 60)
    print("📦 项目打包工具")
    print("=" * 60)
    print(f"📂 输入目录: {input_dir}")
    print(f"📂 输出目录: {output_dir}")
    print()
    
    # 步骤 1: 收集文件和生成目录树
    print("🔍 步骤 1/4: 收集文件和生成目录树...")
    files, tree_lines, dir_counts = collect_files_and_generate_tree(input_dir)
    print(f"   ✅ 共收集到 {len(files)} 个文件")
    
    if not files:
        print("⚠️ 没有找到任何文件，退出程序")
        return
    
    # 步骤 2: 生成 Markdown
    print("\n📝 步骤 2/4: 生成 Markdown 文件...")
    md_content = generate_markdown(files, input_dir, tree_lines, dir_counts)
    md_path = os.path.join(output_dir, 'package.md')
    with open(md_path, 'w', encoding='utf-8') as f:
        f.write(md_content)
    print(f"   ✅ Markdown 文件已保存: {md_path}")
    
    # 步骤 3: 转换为 HTML
    print("\n🌐 步骤 3/4: 转换为 HTML 文件...")
    html_content = convert_to_html(md_content)
    html_path = os.path.join(output_dir, 'package.html')
    with open(html_path, 'w', encoding='utf-8') as f:
        f.write(html_content)
    print(f"   ✅ HTML 文件已保存: {html_path}")
    
    # 步骤 4: 生成截图
    mode = args.screenshot_mode
    mode_labels = {'auto': '自动', 'full': '完整长图', 'segment': '分段截图', 'both': '完整+分段'}
    print(f"\n📸 步骤 4/4: 生成截图（模式: {mode_labels[mode]}）...")
    screenshots_dir = os.path.join(output_dir, 'screenshots')
    os.makedirs(screenshots_dir, exist_ok=True)
    screenshot_path = os.path.join(screenshots_dir, 'package.png')
    
    result_paths = generate_screenshot(html_content, screenshot_path, mode=mode)
    
    # 完成
    print("\n" + "=" * 60)
    print("✨ 打包完成！")
    print("=" * 60)
    print(f"📄 Markdown: {md_path}")
    print(f"🌐 HTML:     {html_path}")
    
    # 显示截图信息
    if os.path.exists(screenshot_path):
        print(f"📸 完整截图: {screenshot_path}")
    segment_dir = os.path.join(screenshots_dir, 'segments')
    if os.path.exists(segment_dir) and len(os.listdir(segment_dir)) > 0:
        print(f"📸 分段截图: {segment_dir}")


if __name__ == '__main__':
    main()
