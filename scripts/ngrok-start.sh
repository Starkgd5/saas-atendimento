#!/bin/bash
# Iniciar ngrok para testes locais

# Instalar ngrok se não tiver
if ! command -v ngrok &> /dev/null; then
    echo "Instalando ngrok..."
    curl -s https://ngrok-agent.s3.amazonaws.com/ngrok.asc | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
    echo "deb https://ngrok-agent.s3.amazonaws.com buster main" | sudo tee /etc/apt/sources.list.d/ngrok.list
    sudo apt update && sudo apt install ngrok
fi

# Iniciar ngrok
ngrok http 8080 --domain=seu-subdominio.ngrok.io