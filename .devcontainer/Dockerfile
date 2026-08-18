FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

# Instalar Zsh e ferramentas
RUN apt-get update && apt-get install -y \
    zsh \
    git \
    curl \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Instalar Go, Python, etc
RUN apt-get update && apt-get install -y \
    golang-go \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Instalar uv
RUN curl -LsSf https://astral.sh/uv/install.sh | sh

# Instalar Volta
RUN curl https://get.volta.sh | bash

USER vscode

# Instalar Oh My Zsh apenas se não existir
RUN if [ ! -d ~/.oh-my-zsh ]; then \
    sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended; \
    fi

# Clonar plugins (verificando se já existem)
RUN git clone https://github.com/zsh-users/zsh-autosuggestions ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions 2>/dev/null || true && \
    git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting 2>/dev/null || true

# Configurar .zshrc apenas se não estiver configurado
RUN if ! grep -q "plugins=(git" ~/.zshrc 2>/dev/null; then \
    echo 'plugins=(git docker docker-compose python pip npm node golang zsh-autosuggestions zsh-syntax-highlighting)' >> ~/.zshrc && \
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && \
    echo 'export PATH="$PATH:/usr/local/go/bin"' >> ~/.zshrc && \
    echo 'export PATH="$VOLTA_HOME/bin:$PATH"' >> ~/.zshrc; \
    fi

USER root