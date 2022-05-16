# Workflows

## Create Personal Access Token (PAT):

- Login on GitHub as vitelabs-bot
- Go to https://github.com/settings/tokens
- Generate new token
- Note: WORKFLOW_PUBLIC_REPO
- Expiration: No expiration
- Select scopes: repo -> public_repo

## Add repo secret

- Go to Settings -> Secrects -> Actions
- New repository secret
- Name: WORKFLOW_PUBLIC_REPO_PAT
- Value: ghp_...
