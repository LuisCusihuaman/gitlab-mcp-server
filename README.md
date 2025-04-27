# GitLab MCP Server 🦊

The GitLab MCP Server is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction)
server that provides seamless integration with GitLab APIs, enabling advanced
automation and interaction capabilities for developers and AI tools within the GitLab ecosystem.

## Use Cases ✨

- Automating GitLab workflows and processes (e.g., managing issues, merge requests).
- Extracting and analyzing data from GitLab projects and groups.
- Building AI-powered tools and applications that interact with GitLab.

## Prerequisites ⚙️

1.  **Docker:** To run the server easily in a container, you need [Docker](https://www.docker.com/) installed and running. *(Alternatively, you can build from source - see below).*
2.  **GitLab Access Token:** You need a GitLab Access Token to authenticate with the API. You can create:
    *   A [Personal Access Token (PAT)](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
    *   A [Project Access Token](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html)
    *   A [Group Access Token](https://docs.gitlab.com/ee/user/group/settings/group_access_tokens.html)
    The required [scopes](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html#available-scopes) depend on the tools you intend to use (e.g., `api` scope grants broad access, or select more granular scopes like `read_repository`, `write_repository`, `read_api`). Grant only the permissions you feel comfortable giving your AI tools.

## Installation 🚀

### Usage with VS Code (Agent Mode)

Add the following JSON block to your User Settings (JSON) file (`Preferences: Open User Settings (JSON)` or `Ctrl+Shift+P`). This configures VS Code to run the server using Docker when Agent Mode is activated.

```json
{
  "mcp": {
    "inputs": [
      {
        "type": "promptString",
        "id": "gitlab_token",
        "description": "GitLab Access Token (PAT, Project, or Group)",
        "password": true
      },
      {
        "type": "promptString",
        "id": "gitlab_host",
        "description": "GitLab Host (e.g., gitlab.com or self-managed URL, leave empty for gitlab.com)",
        "password": false
      }
    ],
    "servers": {
      "gitlab": {
        // Replace with the actual image path once published (e.g., registry.gitlab.com/your-group/gitlab-mcp-server)
        "command": "docker",
        "args": [
          "run",
          "-i",
          "--rm",
          "-e", "GITLAB_TOKEN",
          "-e", "GITLAB_HOST",
          // IMPORTANT: Update this image path when available!
          "your-docker-registry/gitlab-mcp-server:latest" 
        ],
        "env": {
          "GITLAB_TOKEN": "${input:gitlab_token}",
          "GITLAB_HOST": "${input:gitlab_host}"
        }
      }
    }
  }
}
```

*(Note: Replace `"your-docker-registry/gitlab-mcp-server:latest"` with the actual published image path once available.)*

You can also add a similar configuration (without the top-level `mcp` key) to a `.vscode/mcp.json` file in your workspace to share the setup.

More about using MCP server tools in VS Code's [agent mode documentation](https://code.visualstudio.com/docs/copilot/chat/mcp-servers).

### Usage with Claude Desktop

*(Example structure, adapt as needed for Claude Desktop configuration)*

```json
{
  "mcpServers": {
    "gitlab": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e", "GITLAB_TOKEN=<YOUR_TOKEN>",
        "-e", "GITLAB_HOST=<YOUR_GITLAB_URL_OR_EMPTY>", 
        // IMPORTANT: Update this image path when available!
        "your-docker-registry/gitlab-mcp-server:latest" 
      ]
      // Alternatively use env block if Claude supports it:
      // "env": {
      //   "GITLAB_TOKEN": "<YOUR_TOKEN>",
      //   "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>"
      // }
    }
  }
}
```

### Build from source

If you prefer not to use Docker, you can build the binary directly:

1.  Clone the repository.
2.  Navigate to the repository root.
3.  Build the server: `go build -o gitlab-mcp-server ./cmd/gitlab-mcp-server`
4.  Configure your MCP client (e.g., VS Code User Settings JSON) to use the built executable:

```json
// Example for VS Code User Settings (JSON)
{
  "mcp": {
    "servers": {
      "gitlab": {
        "command": "/path/to/your/gitlab-mcp-server", // Replace with actual path
        "args": ["stdio"],
        "env": {
          "GITLAB_TOKEN": "<YOUR_TOKEN>",
          "GITLAB_HOST": "<YOUR_GITLAB_URL_OR_EMPTY>" 
        }
      }
    }
  }
}
```

## Tool Configuration 🛠️

The GitLab MCP Server supports enabling or disabling specific groups of functionalities (toolsets) via the `--toolsets` flag or the `GITLAB_TOOLSETS` environment variable. This allows fine-grained control over the GitLab API capabilities exposed to your AI tools. Enabling only necessary toolsets can improve LLM tool selection and reduce context size.

### Available Toolsets

The following sets of tools are planned (all enabled by default if `GITLAB_TOOLSETS` is not set or set to `"all"`):

| Toolset         | Description                                                                   |
|-----------------|-------------------------------------------------------------------------------|
| `projects`      | Project details, repository operations (files, branches, commits, tags).       |
| `issues`        | Issue management (CRUD, comments, labels, milestones).                       |
| `merge_requests`| Merge request operations (CRUD, comments, approvals, diffs, status checks).  |
| `security`      | Accessing security scan results (SAST, Secret Detection, etc.).                |
| `users`         | User information lookup, potentially current user details.                     |
| `search`        | Utilizing GitLab's scoped search capabilities (projects, issues, MRs, code). |
| *(Potential Future: `ci_cd`, `groups`, `epics`)*                                               |

#### Specifying Toolsets

Pass an allow-list of desired toolsets (comma-separated):

1.  **Using Command Line Argument** (when running binary directly):
    ```bash
    ./gitlab-mcp-server stdio --toolsets issues,merge_requests,projects
    ```

2.  **Using Environment Variable**:
    ```bash
    export GITLAB_TOOLSETS="issues,merge_requests,projects"
    ./gitlab-mcp-server stdio 
    ```
    *(The environment variable `GITLAB_TOOLSETS` takes precedence over the flag.)*

### Using Toolsets With Docker

Pass the toolsets via environment variables when running the container:

```bash
docker run -i --rm \
  -e GITLAB_TOKEN=<your-token> \
  -e GITLAB_HOST=<your-gitlab-url_or_empty> \
  -e GITLAB_TOOLSETS="issues,merge_requests,projects" \
  your-docker-registry/gitlab-mcp-server:latest # IMPORTANT: Update image path!
```

### The "all" Toolset

Use the special value `all` to explicitly enable all available toolsets:

```bash
./gitlab-mcp-server stdio --toolsets all 
# or
export GITLAB_TOOLSETS="all" 
./gitlab-mcp-server stdio
# or with Docker
docker run -i --rm -e GITLAB_TOKEN=... -e GITLAB_TOOLSETS="all" ...
```

## Dynamic Tool Discovery 💡

*(This feature might be implemented later, following the pattern from github-mcp-server)*

Instead of starting with a fixed set of enabled tools, dynamic toolset discovery allows the MCP host (like VS Code or Claude) to list available toolsets and enable them selectively in response to user needs. This can prevent overwhelming the language model with too many tools initially.

### Using Dynamic Tool Discovery

If implemented, enable it via:

*   **Flag:** `./gitlab-mcp-server stdio --dynamic-toolsets`
*   **Environment Variable:** `export GITLAB_DYNAMIC_TOOLSETS=1`
*   **Docker:** `docker run -i --rm -e GITLAB_TOKEN=... -e GITLAB_DYNAMIC_TOOLSETS=1 ...`

When enabled, the server initially exposes only minimal tools, including tools to list and enable other toolsets dynamically.

## GitLab Self-Managed Instances 🏢

To connect to a self-managed GitLab instance instead of `gitlab.com`, use the `--gitlab-host` flag or the `GITLAB_HOST` environment variable. Provide the base URL of your instance (e.g., `https://gitlab.example.com`).

*   **Flag:** `./gitlab-mcp-server stdio --gitlab-host https://gitlab.example.com`
*   **Environment Variable:** `export GITLAB_HOST="https://gitlab.example.com"`
*   **Docker:** `docker run -i --rm -e GITLAB_TOKEN=... -e GITLAB_HOST="https://gitlab.example.com" ...`

If the variable/flag is empty or omitted, the server defaults to `https://gitlab.com`.

## i18n / Overriding Descriptions 🌍

Tool names and descriptions can be customized or translated. Create a `gitlab-mcp-server-config.json` file in the *same directory* as the server binary (or mount it into the container).

The file should contain a JSON object mapping internal translation keys (which correspond to tool names/descriptions) to your desired strings.

**Example `gitlab-mcp-server-config.json`:**
```json
{
  "TOOL_GET_ISSUE_DESCRIPTION": "Fetch details for a specific GitLab issue.",
  "TOOL_CREATE_MERGE_REQUEST_USER_TITLE": "Open New MR"
}
```

You can generate a template file containing all current translation keys by running the server with the `--export-translations` flag:

```bash
./gitlab-mcp-server --export-translations 
# This will create/update gitlab-mcp-server-config.json
```
This flag preserves existing overrides while adding any new keys introduced in the server.

## Contributing & License 🤝

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

This project is released under the [MIT License](LICENSE). 