#!/bin/bash

# Cleanup script for testing install.sh
# This script removes menv installations from common locations

set -e

echo "🧹 Cleaning up previous menv installations..."

# Common install locations
INSTALL_LOCATIONS=(
    "/usr/local/bin/menv"
    "$HOME/.local/bin/menv" 
    "$HOME/bin/menv"
    "/tmp/menv-install-*"
)

# Remove menv from install locations
for location in "${INSTALL_LOCATIONS[@]}"; do
    if [[ "$location" == *"*"* ]]; then
        # Handle wildcards with find
        find /tmp -name "menv-install-*" -type d 2>/dev/null | while read -r dir; do
            if [[ -d "$dir" ]]; then
                echo "  Removing directory: $dir"
                rm -rf "$dir"
            fi
        done
    elif [[ -f "$location" ]]; then
        echo "  Removing: $location"
        rm -f "$location"
    fi
done

# Remove from PATH if it was added (check common shell configs)
SHELL_CONFIGS=(
    "$HOME/.bashrc"
    "$HOME/.zshrc"
    "$HOME/.profile"
    "$HOME/.bash_profile"
)

for config in "${SHELL_CONFIGS[@]}"; do
    if [[ -f "$config" ]] && grep -q "\.local/bin.*menv" "$config" 2>/dev/null; then
        echo "  Found menv PATH reference in $config"
        echo "  Please manually remove any menv-related PATH exports from $config"
    fi
done

# Clear any cached command locations
hash -r 2>/dev/null || true

echo "✅ Cleanup completed!"
echo ""
echo "Verification:"
if command -v menv >/dev/null 2>&1; then
    echo "❌ menv is still available in PATH: $(which menv)"
    echo "   Manual removal may be needed"
else
    echo "✅ menv is no longer available in PATH"
fi