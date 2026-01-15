---
title: Automatic Deployments from GitHub
description: Automatically create new versions and update customer deployments whenever you push a release to GitHub using the Distr GitHub Action.
slug: docs/guides/automatic-deployments-from-github
sidebar:
  order: 7
---

In this guide, you'll learn how to set up a fully automated deployment pipeline that creates new application versions in Distr and automatically updates all your customer deployments whenever you push a new release to GitHub.

By the end of this guide, you'll have:

- A Docker or Helm application configured in Distr
- A GitHub Actions workflow that runs on every release tag
- Automatic version creation in Distr
- Automatic deployment updates to all customer environments

This is particularly useful for SaaS applications where you want to roll out updates to all customers simultaneously, or for managed services where you control the deployment timing.

## Prerequisites

Before starting, ensure you have:

1. A Distr account with an organization set up
2. A GitHub repository containing your application
3. A Docker Compose file (for Docker apps) or Helm chart details (for Helm apps)
4. At least one deployment target configured in Distr (for testing automatic updates)

## Step 1: Create Your Application in Distr

First, you need to create an application in Distr that will receive the automated version updates.

### For Docker Applications

1. Navigate to the **Applications** section in the Distr web interface
2. Click **Add application** in the top right corner
3. Enter a name for your application (e.g., "My SaaS App")
4. Select **Docker** as the application type
5. Click **Create Application**

### For Helm Applications

1. Navigate to the **Applications** section in the Distr web interface
2. Click **Add application** in the top right corner
3. Enter a name for your application (e.g., "My Kubernetes App")
4. Select **Kubernetes** as the application type
5. Click **Create Application**

### Copy the Application ID

After creating the application, you'll see the application details page. At the top, you'll find the **Application ID** (a UUID like `7fa566b3-a20e-4b09-814c-5193c1469f7c`).

**Copy this ID** - you'll need it later for the GitHub Action configuration.

## Step 2: Create a Personal Access Token

The GitHub Action needs to authenticate with Distr to create new versions and update deployments. You'll use a Personal Access Token (PAT) for this.

1. In the Distr web interface, click on your user icon in the top right corner
2. Select **Personal Access Tokens** from the dropdown menu
3. Click **Create token** in the top right corner
4. Enter a descriptive name (e.g., "GitHub Actions - My App")
5. Set an expiry date (recommended: 1 year from now)
6. Click **Create**
7. **Copy the token immediately** - this is the only time you'll see it

For detailed instructions with screenshots, see the [Creating a Personal Access Token](/docs/integrations/personal-access-token/) guide.

## Step 3: Configure GitHub Secrets and Variables

Now you'll store the PAT and Application ID securely in your GitHub repository.

### Add the Personal Access Token as a Secret

GitHub Secrets are encrypted and perfect for storing sensitive information like API tokens.

1. Go to your GitHub repository
2. Click on **Settings** (in the repository toolbar)
3. In the left sidebar, navigate to **Secrets and variables** → **Actions**
4. Click **New repository secret**
5. Name: `DISTR_API_TOKEN`
6. Value: Paste the Personal Access Token you copied earlier
7. Click **Add secret**

### Add the Application ID as a Variable

GitHub Variables are used for non-sensitive configuration values.

1. Stay on the same page (Secrets and variables → Actions)
2. Click on the **Variables** tab
3. Click **New repository variable**
4. Name: `DISTR_APPLICATION_ID`
5. Value: Paste the Application ID you copied earlier
6. Click **Add variable**

## Step 4: Prepare Your Application Files

Ensure your repository contains the necessary files for deployment.

### For Docker Applications

Your repository should include:

- **docker-compose.yaml** - Your Docker Compose file (can be in a subdirectory like `deploy/`)
- **env.template** (optional) - A template file showing which environment variables customers need to configure

Example docker-compose.yaml:

```yaml
services:
  web:
    image: ghcr.io/yourorg/your-app:${VERSION}
    ports:
      - '8080:8080'
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - SECRET_KEY=${SECRET_KEY}
```

Example env.template:

```
# Database connection string
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Secret key for encryption (generate a random string)
SECRET_KEY=
```

### For Helm Applications

Your Helm chart should be published to a registry (OCI or traditional Helm repository). You'll reference it in the GitHub Action.

Optionally, include:

- **base-values.yaml** - Default values for your Helm chart
- **template.yaml** (optional) - Template for customer-specific value overrides

## Step 5: Automate Version Management with Release Please (Recommended)

For Docker applications that build and push container images, you need to ensure your Docker Compose file references the correct image tags for each release. Manually updating these tags is error-prone and tedious. **Release Please** automates this process.

### Why Release Please is Essential

When you release a new version of your application:

1. Your CI/CD builds and pushes Docker images tagged with the release version (e.g., `0.2.0`)
2. Your `docker-compose.yaml` must reference these exact image tags
3. The Distr GitHub Action uploads this compose file to Distr

**Without automation**, you'd need to manually update every image tag in your compose file before each release. Release Please solves this by automatically updating version numbers throughout your repository when creating a release.

### How It Works

Release Please follows this workflow:

1. **Analyzes commits** - Reads your conventional commits (`feat:`, `fix:`, `chore:`, etc.) since the last release
2. **Creates a Release PR** - Opens a pull request with:
   - Updated version numbers in all configured files
   - Generated CHANGELOG.md entries
   - Bumped version following semantic versioning rules
3. **Creates releases** - When you merge the Release PR:
   - Creates a Git tag (e.g., `0.2.0`)
   - Publishes a GitHub release with the changelog
   - Triggers your build and deployment workflows

### Setting Up Release Please

#### 1. Create the Release Please Workflow

Create `.github/workflows/release-please.yaml`:

```yaml
name: Release Please

on:
  push:
    branches:
      - main

jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
```

#### 2. Create the Configuration File

Create `release-please-config.json` in your repository root:

```json
{
  "packages": {
    ".": {
      "release-type": "go",
      "extra-files": ["deploy/docker-compose.yaml"]
    }
  },
  "include-v-in-tag": false
}
```

**Key settings:**

- **`release-type`** - Set to `"go"`, `"node"`, or `"simple"` based on your project
- **`extra-files`** - Lists additional files to update with version numbers
- **`include-v-in-tag`** - Set to `false` to create tags like `0.2.0` instead of `v0.2.0`

For projects with multiple services, add all compose files or package files to `extra-files`.

#### 3. Create the Manifest File

Create `.release-please-manifest.json`:

```json
{
  ".": "0.1.0"
}
```

This tracks your current version. Release Please will update this file automatically.

#### 4. Mark Version Locations in Your Docker Compose File

Add `# x-release-please-version` comments to lines where versions should be updated:

```yaml
services:
  backend:
    image: ghcr.io/yourorg/your-app/backend:0.1.0 # x-release-please-version
    # ... rest of config

  frontend:
    image: ghcr.io/yourorg/your-app/frontend:0.1.0 # x-release-please-version
    # ... rest of config

  proxy:
    image: ghcr.io/yourorg/your-app/proxy:0.1.0 # x-release-please-version
    # ... rest of config
```

Release Please will automatically update these version numbers when creating a release.

### The Complete Release Flow

Here's how everything works together:

1. **Development** - You push commits to `main` using conventional commit messages
2. **Release PR Created** - Release Please automatically:
   - Creates/updates a release PR
   - Updates version numbers in `docker-compose.yaml`
   - Generates changelog entries
3. **Merge Release PR** - When you merge the PR:
   - Git tag is created (e.g., `0.2.0`)
   - GitHub release is published
4. **Build Images** - Tag triggers your build workflows:
   - Docker images are built and pushed with the version tag
5. **Push to Distr** - Tag triggers the `push-distr.yaml` workflow:
   - **IMPORTANT**: This workflow must wait for all Docker images to be pushed before running
   - Creates new version in Distr with the updated compose file (now referencing the correct image tags)
   - Optionally updates all deployments

### Example from hello-distr

The [hello-distr](https://github.com/distr-sh/hello-distr) repository demonstrates this complete setup:

**docker-compose.yaml:**

```yaml
services:
  backend:
    image: ghcr.io/distr-sh/hello-distr/backend:0.2.1 # x-release-please-version
  frontend:
    image: ghcr.io/distr-sh/hello-distr/frontend:0.2.1 # x-release-please-version
  proxy:
    image: ghcr.io/distr-sh/hello-distr/proxy:0.2.1 # x-release-please-version
```

**Build workflow** (`.github/workflows/build-backend.yaml`):

```yaml
on:
  push:
    tags:
      - '*'

jobs:
  build:
    steps:
      - uses: docker/metadata-action@v5
        with:
          images: ghcr.io/distr-sh/hello-distr/backend
          tags: |
            type=semver,pattern={{version}}
      - uses: docker/build-push-action@v6
        with:
          push: ${{ startsWith(github.ref, 'refs/tags/') }}
          tags: ${{ steps.meta.outputs.tags }}
```

When version `0.2.0` is released:

1. Release Please updates all image tags to `0.2.0`
2. Git tag `0.2.0` is created
3. Build workflows push images tagged `0.2.0`
4. Distr workflow creates version with compose file referencing `0.2.0` images

**Note:** In hello-distr's current setup, build workflows and the Distr workflow trigger simultaneously on tags. In practice, image builds take longer than uploading a compose file, so this works. However, for production use, you should implement proper workflow sequencing to guarantee images are available before deployments are triggered.

## Step 6: Create the GitHub Actions Workflow

Now you'll create a workflow that runs automatically whenever you create a new release tag.

1. In your repository, create the directory `.github/workflows/` (if it doesn't exist)
2. Create a new file: `.github/workflows/push-distr.yaml`

### For Docker Applications

```yaml
name: Push Distr Application Version

on:
  push:
    tags:
      - '*'

jobs:
  push-to-distr:
    name: Create Version and Update Deployments
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Create Distr Version and Update Deployments
        uses: distr-sh/distr-create-version-action@v1
        with:
          api-token: ${{ secrets.DISTR_API_TOKEN }}
          application-id: ${{ vars.DISTR_APPLICATION_ID }}
          version-name: ${{ github.ref_name }}
          compose-file: ${{ github.workspace }}/docker-compose.yaml
          template-file: ${{ github.workspace }}/env.template
          link-template: 'http://{{ .Env.APP_HOST }}'
          update-deployments: true
```

### For Helm Applications

```yaml
name: Push Distr Application Version

on:
  push:
    tags:
      - '*'

jobs:
  push-to-distr:
    name: Create Version and Update Deployments
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Create Distr Version and Update Deployments
        uses: distr-sh/distr-create-version-action@v1
        with:
          api-token: ${{ secrets.DISTR_API_TOKEN }}
          application-id: ${{ vars.DISTR_APPLICATION_ID }}
          version-name: ${{ github.ref_name }}
          chart-type: 'oci'
          chart-url: oci://ghcr.io/yourorg/your-chart
          chart-version: ${{ github.ref_name }}
          base-values-file: ${{ github.workspace }}/base-values.yaml
          template-file: ${{ github.workspace }}/template.yaml
          link-template: 'https://{{ .Env.INGRESS_HOST }}'
          update-deployments: true
```

### Key Configuration Options

- **`api-token`** - Your Personal Access Token (from GitHub Secrets)
- **`application-id`** - Your Application ID (from GitHub Variables)
- **`version-name`** - The version name (here we use the git tag name)
- **`link-template`** - Template for generating links to deployments (e.g., `http://{{ .Env.APP_HOST }}`). See [Application Links](/docs/guides/application-links/) for details
- **`update-deployments: true`** - **This is the key setting that enables automatic deployment updates**

When `update-deployments` is set to `true`, the action will:

1. Create the new version in Distr
2. Find all deployment targets where this application is deployed
3. Update each deployment to the new version
4. Skip targets that are already on the new version or don't have the app deployed

**IMPORTANT**: If your application builds and pushes Docker images, the Distr workflow **must wait** for all images to be pushed before running. Otherwise, deployments will fail because the images referenced in your `docker-compose.yaml` won't be available yet.

```yaml
name: Build and Push to Distr

on:
  push:
    tags:
      - '*'

jobs:
  build-backend:
    runs-on: ubuntu-latest
    steps:
      # ... build and push backend image

  build-frontend:
    runs-on: ubuntu-latest
    steps:
      # ... build and push frontend image

  build-proxy:
    runs-on: ubuntu-latest
    steps:
      # ... build and push proxy image

  push-to-distr:
    needs: [build-backend, build-frontend, build-proxy]
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Create Distr Version and Update Deployments
        uses: distr-sh/distr-create-version-action@v1
        with:
          api-token: ${{ secrets.DISTR_API_TOKEN }}
          application-id: ${{ vars.DISTR_APPLICATION_ID }}
          version-name: ${{ github.ref_name }}
          compose-file: ${{ github.workspace }}/deploy/docker-compose.yaml
          template-file: ${{ github.workspace }}/deploy/env.template
          update-deployments: true
```

The `needs:` clause ensures `push-to-distr` only runs after all build jobs complete.

## Step 7: Test Your Automation

Now it's time to test the complete automation pipeline.

### Create a Test Deployment Target

Before testing, make sure you have at least one deployment target configured:

1. Navigate to **Deployments** in the Distr web interface
2. Click **Add deployment** to create a test deployment target
3. Follow the wizard to set up a Docker or Kubernetes deployment target
4. Deploy an earlier version of your application to this target

### Create a Release

#### If Using Release Please (Recommended)

1. Commit and push your changes using conventional commits:

   ```bash
   git add .github/workflows/
   git commit -m "feat: add automated Distr deployment workflow"
   git push origin main
   ```

2. Release Please will automatically create a Release PR. Review and merge it.

3. When you merge the Release PR, Release Please will:
   - Create a Git tag (e.g., `0.1.0`)
   - Trigger your build workflows (which push Docker images)
   - Trigger the Distr workflow (which creates the version and updates deployments)

#### If Using Manual Tags

1. Commit and push your workflow file to GitHub:

   ```bash
   git add .github/workflows/push-distr.yaml
   git commit -m "Add automated Distr deployment workflow"
   git push
   ```

2. Create and push a release tag:
   ```bash
   git tag 0.1.0
   git push origin 0.1.0
   ```

### Monitor the Workflow

1. Go to your GitHub repository
2. Click on the **Actions** tab
3. You should see your workflow running
4. Click on the workflow run to see detailed logs

The workflow will:

- Check out your code
- Create a new version (e.g., `0.1.0`) in Distr
- Update all deployments to the new version
- Show which deployment targets were updated and which were skipped

### Verify in Distr

1. In the Distr web interface, navigate to your application
2. You should see the new version in the version list
3. Navigate to **Deployments**
4. Your deployment target should now show the new version as the deployed version

## Understanding Automatic Updates

When `update-deployments: true` is enabled, the GitHub Action will:

### Update These Targets

- Deployment targets that have your application deployed
- Deployment targets on an older version than the newly created one
- Deployment targets that are online and reachable

### Skip These Targets

- Deployment targets that don't have your application deployed
- Deployment targets already on the target version
- The action will log why each target was skipped

### Example Output

```
Updating all deployments to the new version...
Updated 3 deployment target(s)
Skipped 2 deployment target(s):
  - Customer A Production: Already on target version
  - Customer B Testing: Application not deployed on this target
```

## Troubleshooting

### Workflow Fails with "401 Unauthorized"

**Problem:** The GitHub Action cannot authenticate with Distr.

**Solution:**

- Verify your `DISTR_API_TOKEN` secret is set correctly
- Check that the token hasn't expired
- Ensure the token has the necessary permissions
- If using self-hosted Distr, verify the `api-base` URL is correct

### Workflow Succeeds but No Deployments Update

**Problem:** Version is created but deployments aren't updated.

**Solution:**

- Verify `update-deployments: true` is set in the workflow
- Check that deployment targets have the application deployed
- Ensure deployment targets are online and reachable
- Review the workflow logs for "Skipped" messages explaining why targets weren't updated

### Deployments Update but Application Doesn't Start

**Problem:** Deployment shows updated but the application fails to start.

**Solution:**

- Check the deployment status in Distr for error messages
- Verify your Docker Compose file or Helm chart is valid
- Ensure required environment variables are configured in the deployment
- Check the deployment logs in the Distr web interface

### Deployments Fail with "Image Pull Error" or "manifest unknown"

**Problem:** Deployments fail immediately after creation with errors like "failed to pull image" or "manifest for image not found".

**Cause:** The Distr workflow ran before your Docker images were fully pushed to the registry.

**Solution:**

This is a critical sequencing issue. In summary:

1. Use `workflow_run` to make the Distr workflow wait for build workflows to complete
2. Use `needs:` in a combined workflow to enforce job order
3. Set `update-deployments: false` and update manually after images are available

**Quick check:** Look at your GitHub Actions runs - do the build workflows complete _after_ the Distr workflow? If yes, you need to fix the sequencing.

### Version Already Exists Error

**Problem:** GitHub Action fails because the version name already exists.

**Solution:**

- Each version name must be unique within an application
- If re-running a workflow for the same tag, delete the existing version in Distr first
- Consider using more specific version names (e.g., include build numbers or timestamps)

## Next Steps

Now that you have automatic deployments set up, consider:

- **[Application Licenses](/docs/guides/application-licenses/)** - Control which customers receive automatic updates
- **[Application Links](/docs/guides/application-links/)** - Create dynamic links for customers to access their deployments
- **[Distr SDK](/docs/integrations/sdk/)** - Build custom automation and integrations
- **[Distr API](/docs/integrations/api/)** - Explore advanced API capabilities

## Additional Resources

- [distr-create-version-action GitHub Repository](https://github.com/distr-sh/distr-create-version-action)
- [hello-distr Example Application](https://github.com/distr-sh/hello-distr) - Complete example with Release Please, Docker builds, and automatic deployments
- [Release Please Documentation](https://github.com/googleapis/release-please) - Automated release management
- [Conventional Commits Specification](https://www.conventionalcommits.org/) - Commit message format for Release Please
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Distr Discord Community](https://discord.gg/6qqBSAWZfW)

---

Have questions? Join our [Discord community](https://discord.gg/6qqBSAWZfW) or check out the [FAQs](/docs/faqs/).
