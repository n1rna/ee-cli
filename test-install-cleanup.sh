#!/bin/bash

# Cleanup script for testing install.sh
# This script removes ee installations from common locations

set -e

echo "🧹 Cleaning up previous ee installations..."

# Common install locations
INSTALL_LOCATIONS=(
    "/usr/local/bin/ee"
    "$HOME/.local/bin/ee"
    "$HOME/bin/ee"
    "/tmp/ee-install-*"
)

# Remove ee from install locations
for location in "${INSTALL_LOCATIONS[@]}"; do
    if [[ "$location" == *"*"* ]]; then
        # Handle wildcards with find
        find /tmp -name "ee-install-*" -type d 2>/dev/null | while read -r dir; do
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
    if [[ -f "$config" ]] && grep -q "\.local/bin.*ee" "$config" 2>/dev/null; then
        echo "  Found ee PATH reference in $config"
        echo "  Please manually remove any ee-related PATH exports from $config"
    fi
done

# Clear any cached command locations
hash -r 2>/dev/null || true

echo "✅ Cleanup completed!"
echo ""
echo "Verification:"
if command -v ee >/dev/null 2>&1; then
    echo "❌ ee is still available in PATH: $(which ee)"
    echo "   Manual removal may be needed"
else
    echo "✅ ee is no longer available in PATH"
fi