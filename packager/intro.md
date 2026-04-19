# Packager 工具详解

## 1. 工具简介

Packager 是一个项目打包工具，用于自动汇总指定目录下的所有文件，生成目录树总览、文件统计信息，并将所有文件内容整理到一个完整的 Markdown 文件中，同时转换为 HTML 并生成截图。

**主要功能**：
- 自动收集目录下的所有文件（排除指定目录和文件）
- 生成带图标的目录树总览
- 统计各目录下的文件数量
- 按目录分组整理文件内容到 Markdown
- 转换 Markdown 为带样式的 HTML
- 使用 Playwright 生成 HTML 页面的完整截图

## 2. 目录结构

```
packager/
├── main.py           # 主脚本（核心功能实现）
├── run.sh           # 一键执行脚本
├── requirements.txt # 依赖文件
├── .venv/           # 虚拟环境（自动创建）
└── output/          # 输出目录（自动创建）
    ├── package.md   # 生成的 Markdown 文件
    ├── package.html # 生成的 HTML 文件
    └── package.png  # 生成的截图（需要安装 Playwright 浏览器）
```

## 3. 核心功能详解

### 3.1 文件收集和过滤

`main.py` 中的 `collect_files` 函数负责收集目录下的所有文件，并过滤掉不需要的目录和文件：

- **排除目录**：`venv`, `.venv`, `__pycache__`, `.git`, `.idea`, `node_modules` 等
- **排除文件**：`.DS_Store`, `Thumbs.db`, `.gitignore`, `.env` 等
- **排序**：按相对路径排序，确保输出顺序一致

### 3.2 目录树生成

`generate_directory_tree` 函数生成带图标的目录树结构：

- **图标**：根据文件类型显示不同的图标（如 🐍 代表 Python 文件，📝 代表 Markdown 文件）
- **缩进**：根据目录层级显示不同的缩进
- **排序**：文件按名称排序，确保输出美观

### 3.3 目录文件数统计

`count_files_by_directory` 函数统计各目录下的文件数量：
- 按目录路径分组
- 生成表格形式的统计结果

### 3.4 Markdown 生成

`generate_markdown` 函数生成完整的 Markdown 内容，包括：

1. **基本信息**：生成时间、源目录、文件总数
2. **目录树总览**：带图标的树形结构
3. **文件数统计**：各目录的文件数量表格
4. **文件内容详情**：按目录分组展示所有文件内容，包括：
   - 文件路径
   - 文件内容（使用代码块，自动识别语言）
   - 错误处理（无法读取的文件显示错误信息）

### 3.5 HTML 转换

`convert_to_html` 函数将 Markdown 转换为带样式的 HTML：

- 使用 `markdown` 库进行转换
- 支持代码高亮、表格等扩展
- 添加响应式样式，适配不同屏幕尺寸
- 美观的界面设计，包括标题、代码、表格样式

### 3.6 截图生成

`generate_screenshot` 函数使用 Playwright 生成 HTML 页面的完整截图：

- 启动无头 Chrome 浏览器
- 加载 HTML 内容
- 等待页面加载完成
- 截取整个页面

## 4. 一键执行脚本

`run.sh` 脚本提供了便捷的一键执行功能：

1. **自动创建虚拟环境**：如果 `.venv` 目录不存在，自动创建
2. **激活虚拟环境**：确保在虚拟环境中执行命令
3. **安装依赖**：自动安装 `requirements.txt` 中的依赖
4. **执行打包**：调用 `main.py` 执行打包功能
5. **默认配置**：默认打包 soulmark 项目根目录，输出到 `output` 目录

## 5. 使用方法

### 5.1 基本使用

```bash
# 进入 packager 目录
cd packager

# 一键执行（默认打包 soulmark 目录）
./run.sh
```

### 5.2 自定义输入/输出目录

```bash
# 打包指定目录
./run.sh -i /path/to/project

# 指定输出目录
./run.sh -i /path/to/project -o /path/to/output

# 显示帮助信息
./run.sh -h
```

### 5.3 安装 Playwright 浏览器（用于截图）

```bash
# 进入 packager 目录
cd packager

# 激活虚拟环境
source .venv/bin/activate

# 安装 Playwright 浏览器
playwright install chromium
```

## 6. 技术实现细节

### 6.1 模块结构

`main.py` 采用模块化设计，主要分为以下几个部分：

1. **配置常量**：定义需要排除的目录和文件，以及文件扩展名到语言的映射
2. **文件收集和过滤**：实现文件的收集、过滤和排序
3. **目录统计**：实现目录树生成和文件数统计
4. **Markdown 生成**：实现 Markdown 内容的生成
5. **HTML 转换**：实现 Markdown 到 HTML 的转换
6. **截图生成**：实现 HTML 页面的截图
7. **主程序**：解析命令行参数，执行打包流程

### 6.2 核心函数

| 函数名 | 功能 | 参数 | 返回值 |
|-------|------|------|-------|
| `collect_files` | 收集目录下的所有文件 | `input_dir` (输入目录路径) | `list` (文件列表) |
| `generate_directory_tree` | 生成目录树结构 | `input_dir` (输入目录路径) | `str` (目录树字符串) |
| `count_files_by_directory` | 统计各目录下的文件数 | `files` (文件列表), `input_dir` (输入目录路径) | `dict` (目录到文件数的映射) |
| `generate_markdown` | 生成 Markdown 内容 | `files` (文件列表), `input_dir` (输入目录路径) | `str` (Markdown 内容) |
| `convert_to_html` | 转换 Markdown 为 HTML | `md_content` (Markdown 内容) | `str` (HTML 内容) |
| `generate_screenshot` | 生成 HTML 截图 | `html_content` (HTML 内容), `output_path` (输出路径) | `bool` (是否成功) |

### 6.3 依赖说明

| 依赖 | 版本 | 用途 |
|------|------|------|
| `markdown` | 3.9 | Markdown 到 HTML 的转换 |
| `playwright` | 1.58.0 | 生成 HTML 页面的截图 |

## 7. 输出示例

### 7.1 Markdown 输出

生成的 Markdown 文件包含以下部分：

```markdown
# 📦 项目打包内容
**生成时间**: 2026-04-12 15:38:28
**源目录**: `/Users/qi/Documents/trae_projects/soulmark`
**文件总数**: 7

---
## 📂 目录树总览
```
📁 soulmark/
  📝 README.md
  📁 claude_code/
    📝 claude_code_research.md
  📁 packager/
    🐍 main.py
    📄 requirements.txt
    📄 run.sh
    📁 output/
      🌐 package.html
      📝 package.md
```

---
## 📊 各目录文件数统计

| 目录路径 | 文件数 |
|----------|--------|
| `[根目录]` | 1 |
| `claude_code` | 1 |
| `packager` | 3 |
| `packager/output` | 2 |

---
## 📄 文件内容详情

### 📁 [根目录]

#### `README.md`
> 路径: `README.md`

```markdown
## Objectives Soulmark
1. 持续总结和记录学习/工作：沉淀问题背景、过程、结论与可复用要点
2. 以季度为周期成长能力：为每个季度设定 1–2 个能力主题，并复盘结果
3. 建立个人知识体系：按领域维护索引，输出模板、清单与最佳实践
4. 形成反思闭环：对关键项目/决策做复盘，提炼行动项并跟踪关闭
5. 打造技术深度主线：选择长期方向持续专项学习与实践，稳定产出高质量内容
```
```

### 7.2 HTML 输出

生成的 HTML 文件具有以下特点：
- 响应式设计，适配不同屏幕尺寸
- 美观的代码高亮
- 清晰的标题层级
- 友好的表格样式
- 舒适的阅读体验

## 8. 注意事项

1. **截图功能**：需要安装 Playwright 浏览器，否则会跳过截图步骤
2. **性能考虑**：对于大型项目，生成的文件可能会比较大，建议只打包必要的文件
3. **编码问题**：默认使用 UTF-8 编码读取文件，对于其他编码的文件可能会有问题
4. **权限问题**：确保对输入目录有读取权限，对输出目录有写入权限

## 9. 扩展建议

1. **添加文件类型过滤**：允许用户指定只打包特定类型的文件
2. **增加压缩功能**：将生成的文件压缩为 zip 包
3. **添加文件大小统计**：在统计信息中添加文件大小信息
4. **支持自定义模板**：允许用户自定义 Markdown 和 HTML 模板
5. **增加多语言支持**：支持生成英文版本的打包内容

## 10. 总结

Packager 工具是一个功能强大、使用便捷的项目打包工具，它可以帮助开发者快速汇总项目文件，生成清晰的目录结构和文件内容文档。通过一键执行脚本，用户可以轻松完成从文件收集到截图生成的整个过程，大大提高了工作效率。

无论是用于项目文档生成、代码审查还是知识归档，Packager 工具都能提供高质量的输出结果，为用户节省宝贵的时间和精力。