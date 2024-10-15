```markdown
# LazyAI

LazyAI is a command-line interface (CLI) tool designed to streamline your development workflow by integrating with AI services and project management tools. It helps you effortlessly generate commit messages, pull request descriptions, and interact with AI models.

## Installation

To install LazyAI, you need to have Go installed on your machine. Once you have Go set up, you can install LazyAI by running the following command:

```sh
go install github.com/nlgtEA/lazyai@latest
```

Make sure your `$GOPATH/bin` is added to your system's `$PATH` so you can run the `lazyai` command from anywhere.

## Configuration

LazyAI requires a configuration file to store API tokens and other necessary settings. Create a file named `.lazyai.yml` in your home directory with the following structure:

```yaml
skydeck:
  accessToken: <your_access_token>
  refreshToken: <your_refresh_token>
  convoID: <default_conversation_id>

pivotalTracker:
  apiToken: <your_api_token>
  projectID: <your_project_id>
  owner: <your_account_owner_name>
```

Replace the placeholders with your actual tokens and IDs.

## Usage

### Generate a Pull Request Description

To generate a pull request description based on the differences from the base branch, use:

```sh
lazyai pr
```

### Send a Message to SkyDeck

To send a message to the SkyDeck AI service, use:

```sh
lazyai sdchat "Your message here"
```

### Retrieve a Pivotal Tracker Story

To retrieve the description of your active Pivotal Tracker story, use:

```sh
lazyai pickPT
```

### Others
For more details on each command, you can use the `--help` flag:

```sh
lazyai <command> --help
```

## Contributing

Contributions are welcome! Feel free to submit a pull request or report any issues you encounter.

## License

This project is licensed under the MIT License.
```
