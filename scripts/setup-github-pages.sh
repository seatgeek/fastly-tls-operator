#!/bin/bash

# Setup script for GitHub Pages Helm repository
# This script initializes the gh-pages branch for hosting the Helm chart

set -e

echo "🚀 Setting up GitHub Pages for Helm chart repository..."

# Check if we're in the right directory
if [ ! -f "charts/fastly-operator/Chart.yaml" ]; then
    echo "❌ Error: Must be run from the fastly-operator project root"
    exit 1
fi

# Check if gh-pages branch already exists
if git show-ref --quiet refs/heads/gh-pages; then
    echo "✅ gh-pages branch already exists"
else
    echo "📝 Creating gh-pages branch..."
    
    # Create orphan branch for gh-pages
    git checkout --orphan gh-pages
    
    # Remove all files from the branch
    git rm -rf .
    
    # Create initial index.html
    cat > index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Fastly Operator Helm Repository</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 2rem; }
        .container { max-width: 800px; margin: 0 auto; }
        code { background: #f4f4f4; padding: 2px 4px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 1rem; border-radius: 5px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Fastly Operator Helm Repository</h1>
        
        <p>This repository hosts the Helm chart for the Fastly Operator.</p>
        
        <h2>📦 Installation</h2>
        
        <p>Add the Helm repository:</p>
        <pre><code>helm repo add fastly-operator https://seatgeek.github.io/fastly-operator/
helm repo update</code></pre>
        
        <p>Install the Fastly Operator:</p>
        <pre><code>helm install fastly-operator fastly-operator/fastly-operator</code></pre>
        
        <h2>🔗 Links</h2>
        <ul>
            <li><a href="https://github.com/seatgeek/fastly-operator">GitHub Repository</a></li>
            <li><a href="index.yaml">Helm Repository Index</a></li>
        </ul>
        
        <h2>📊 Available Charts</h2>
        <p>See the <a href="index.yaml">index.yaml</a> file for available chart versions.</p>
    </div>
</body>
</html>
EOF

    # Create initial empty index.yaml
    cat > index.yaml << 'EOF'
apiVersion: v1
entries: {}
generated: "2024-01-01T00:00:00Z"
EOF

    # Add and commit initial files
    git add index.html index.yaml
    git commit -m "Initial GitHub Pages setup for Helm repository"
    
    echo "✅ gh-pages branch created successfully"
fi

# Switch back to main branch
git checkout main

echo "🎉 GitHub Pages setup complete!"
echo ""
echo "Next steps:"
echo "1. Push the gh-pages branch: git push origin gh-pages"
echo "2. Enable GitHub Pages in repository settings"
echo "3. Set source to 'gh-pages' branch"
echo "4. Your Helm repository will be available at:"
echo "   https://seatgeek.github.io/fastly-operator/" 