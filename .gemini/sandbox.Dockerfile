FROM gemini-cli-sandbox

RUN apt-get update && \
    apt-get install -y build-essential golang-go git
