FROM gemini-cli-sandbox:latest

RUN apt-get update && \
    apt-get install -y build-essential golang-go git
