# Tutor

Tutor is a Go-based backend project with a svelte-frontend. It's a chat interface with some agentic processes to help aide with learning or practicing a concept.

Prerequisites
- Go 1.18 or higher
- Git
- API keys as required (see Configuration below)

Getting Started
Clone the Repository

```bash
git clone https://github.com/alberrttt/tutor.git
cd tutor
```
Configuration

Create a .env file in the root directory of your project and add any required API keys. Here’s a sample .env file:

plaintext
```
SAMBANOVA_CLOUD_API_KEY=<your_api_key_here>
```
Make sure to replace <your_api_key_here> with your SambaNova Cloud API key.

To start the project, simply run:

```bash
go run .
```
This command will compile and run the project, making it ready to handle API requests.
Usage

Once the server is running, you can interact with it by sending API requests to the appropriate endpoints. Detailed API documentation will be available in future updates.