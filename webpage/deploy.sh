#!/bin/bash

# 清理之前的构建文件
rm -rf build

# 构建项目
npm run build

# 创建一个临时目录用于gh-pages分支
mkdir -p temp_deploy

# 复制构建文件到临时目录
cp -r build/* temp_deploy/

# 切换到临时目录
cd temp_deploy

# 初始化一个新的git仓库
git init
git add .
git commit -m "Deploy to GitHub Pages"

# 强制推送到gh-pages分支
git push -f git@github.com:cyberspacesec/snir-skills.git master:gh-pages

# 返回并清理
cd ..
rm -rf temp_deploy

echo "部署完成! 网站可访问: https://cyberspacesec.github.io/go-snir/" 