#!/bin/bash
# Carregar variáveis de ambiente para o container

if [ -f .env ]; then
    export $(cat .env | grep -v '#' | awk '/=/ {print $1}')
fi