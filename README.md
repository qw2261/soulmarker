## Objectives Soulmark
1. 持续总结和记录学习/工作：沉淀问题背景、过程、结论与可复用要点
2. 以季度为周期成长能力：为每个季度设定 1–2 个能力主题，并复盘结果
3. 建立个人知识体系：按领域维护索引，输出模板、清单与最佳实践
4. 形成反思闭环：对关键项目/决策做复盘，提炼行动项并跟踪关闭
5. 打造技术深度主线：选择长期方向持续专项学习与实践，稳定产出高质量内容

## 项目结构

```
soulmark/
├── claude_code/          # Claude Code 研究笔记
│   └── claude_code_research.md  # Claude Code 源码深度分析
├── packager/             # 项目打包工具
│   ├── main.py           # 主脚本
│   ├── run.sh           # 一键执行脚本
│   ├── requirements.txt # 依赖文件
│   └── intro.md         # 工具使用说明
└── README.md            # 项目说明
```

## 现有项目

### 1. Claude Code 研究
- **文件**：`claude_code/claude_code_research.md`
- **内容**：Claude Code 源码深度分析，包括架构设计、Agent 系统、安全机制等
- **特点**：包含详细的架构图和技术分析

### 2. 项目打包工具 (Packager)
- **目录**：`packager/`
- **功能**：自动汇总项目文件，生成目录树、Markdown、HTML 和截图
- **使用方法**：
  ```bash
  cd packager
  ./run.sh  # 一键执行，默认打包 soulmark 目录
  ```
- **特点**：
  - 智能分段截图，处理长页面
  - 自动安装依赖和浏览器
  - 美观的输出格式

## 学习计划

- [ ] 每周至少完成 1 篇深度总结
- [ ] 每季度进行能力主题复盘
- [ ] 持续扩展知识体系索引