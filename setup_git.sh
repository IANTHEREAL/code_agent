#!/bin/bash
#
# Headless Git & GitHub CLI Authentication Script
#
# This script configures Git and authenticates the GitHub CLI (gh) in a
# non-interactive environment using environment variables.
#
# Required Environment Variables:
#   - GITHUB_TOKEN: A GitHub Personal Access Token with the necessary scopes (e.g., 'repo', 'workflow').
#
# Optional Environment Variables:
#   - GIT_AUTHOR_NAME:  The name to use for Git commits.
#   - GIT_AUTHOR_EMAIL: The email to use for Git commits.
#

# --- Script Safety ---
# Exit immediately if a command exits with a non-zero status.
set -e
# Treat unset variables as an error when substituting.
set -u
# Pipes will fail if any command in the pipe fails, not just the last one.
set -o pipefail

# --- Pre-flight Checks ---
# 1. Check for required commands
if ! command -v git &> /dev/null; then
    echo "Error: git is not installed. Please install it first." >&2
    exit 1
fi
if ! command -v gh &> /dev/null; then
    echo "Error: gh (GitHub CLI) is not installed. Please install it first." >&2
    exit 1
fi

# 2. Check for the required environment variable
if [ -z "${GITHUB_TOKEN}" ]; then
    echo "Error: GITHUB_TOKEN environment variable is not set." >&2
    echo "Please export your GitHub Personal Access Token before running this script." >&2
    exit 1
fi

# --- Main Logic ---
echo "--- 1. Configuring Git Author Information ---"
# Use provided env vars or fall back to a generic GitHub Actions bot identity
GIT_USER="${GIT_AUTHOR_NAME:-github-actions[bot]}"
GIT_EMAIL="${GIT_AUTHOR_EMAIL:-41898282+github-actions[bot]@users.noreply.github.com}"

git config --global user.name "$GIT_USER"
git config --global user.email "$GIT_EMAIL"
echo "✓ Git author set to: $(git config user.name) <$(git config user.email)>"
echo

echo "--- 2. Authenticating GitHub CLI ---"
# Log in non-interactively by piping the token to the --with-token flag.
# This securely configures gh and also sets up Git's credential helper.
echo "${GITHUB_TOKEN}" | gh auth login --with-token
echo "✓ GitHub CLI authentication successful."
echo

echo "--- 3. Verifying Configuration ---"
echo "Verifying gh authentication status..."
gh auth status
echo
echo "Verifying Git credential helper..."
# This should now output 'gh auth git-credential'
echo "Git credential.helper is set to: '$(git config --get credential.helper)'"
echo

echo "✅ Configuration complete. Git and gh are ready to use."
