#!/bin/bash

# NowzFX GitHub上传脚本
# 使用说明：将此脚本复制到项目目录并执行

set -e

echo "🚀 NowzFX GitHub 上传助手"
echo "================================"
echo ""

# 1. 设置Git用户信息（如果未设置）
if [ -z "$(git config user.name)" ]; then
    read -p "请输入您的GitHub用户名: " username
    git config user.name "$username"
    echo "✅ 已设置用户名: $username"
fi

if [ -z "$(git config user.email)" ]; then
    read -p "请输入您的GitHub邮箱: " email
    git config user.email "$email"
    echo "✅ 已设置邮箱: $email"
fi

echo ""
echo "📋 当前Git配置："
echo "   用户名: $(git config user.name)"
echo "   邮箱: $(git config user.email)"
echo ""

# 2. 检查是否已有commit
if ! git log -1 > /dev/null 2>&1; then
    echo "📝 创建初始提交..."
    git add .
    git commit -m "Initial commit: NowzFX AI Trading Platform

- AI-driven cryptocurrency trading system
- Multi-exchange and multi-strategy support  
- Real-time monitoring with web interface
- Docker containerization support
- Secure API key management"
    echo "✅ 提交完成"
fi

echo ""
echo "🔐 现在需要创建GitHub仓库并推送代码"
echo ""
echo "请选择创建方式："
echo "  1) 我已经在GitHub上手动创建了私有仓库 nowzfx"
echo "  2) 使用GitHub CLI自动创建（需要先安装gh）"
echo "  3) 显示手动创建步骤"
echo ""
read -p "请选择 (1/2/3): " choice

case $choice in
    1)
        read -p "请输入您的GitHub用户名: " gh_username
        REPO_URL="https://github.com/$gh_username/nowzfx.git"
        
        echo "🔗 设置远程仓库..."
        git remote remove origin 2>/dev/null || true
        git remote add origin "$REPO_URL"
        
        echo "📤 推送到GitHub..."
        git branch -M main
        git push -u origin main
        
        echo ""
        echo "✅ 成功！您的项目已上传到："
        echo "   https://github.com/$gh_username/nowzfx"
        ;;
        
    2)
        if ! command -v gh &> /dev/null; then
            echo "❌ GitHub CLI (gh) 未安装"
            echo "请先安装: brew install gh"
            echo "然后运行: gh auth login"
            exit 1
        fi
        
        echo "🔑 检查GitHub认证..."
        if ! gh auth status &> /dev/null; then
            echo "需要登录GitHub..."
            gh auth login
        fi
        
        echo "📦 创建私有仓库 nowzfx..."
        gh repo create nowzfx --private --source=. --remote=origin --push
        
        echo ""
        echo "✅ 成功！仓库已创建并推送"
        gh_username=$(gh api user -q .login)
        echo "   https://github.com/$gh_username/nowzfx"
        ;;
        
    3)
        echo ""
        echo "📖 手动创建步骤："
        echo ""
        echo "1. 访问 https://github.com/new"
        echo "2. 填写仓库信息："
        echo "   - Repository name: nowzfx"
        echo "   - Description: AI驱动的加密货币交易平台"
        echo "   - Visibility: 🔒 Private (私有)"
        echo "   - ❌ 不要勾选 'Add a README file'"
        echo "   - ❌ 不要勾选 'Add .gitignore'"
        echo "   - ❌ 不要选择 'Choose a license'"
        echo "3. 点击 'Create repository'"
        echo ""
        echo "4. 创建后，在本地执行："
        echo ""
        read -p "请输入您的GitHub用户名: " gh_username
        echo ""
        echo "   git remote add origin https://github.com/$gh_username/nowzfx.git"
        echo "   git branch -M main"
        echo "   git push -u origin main"
        echo ""
        read -p "是否现在执行推送？(y/n): " do_push
        
        if [ "$do_push" = "y" ] || [ "$do_push" = "Y" ]; then
            REPO_URL="https://github.com/$gh_username/nowzfx.git"
            git remote remove origin 2>/dev/null || true
            git remote add origin "$REPO_URL"
            git branch -M main
            git push -u origin main
            echo "✅ 推送完成！"
        fi
        ;;
        
    *)
        echo "无效选择"
        exit 1
        ;;
esac

echo ""
echo "🎉 完成！"
echo ""
echo "⚠️  重要提醒："
echo "   - 仓库已设置为私有（仅您可见）"
echo "   - .env 文件已被排除，不会上传"
echo "   - 请勿在代码中硬编码API密钥"
echo ""
