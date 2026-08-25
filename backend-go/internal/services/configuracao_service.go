package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type ConfiguracaoService struct {
	repo *repository.ConfiguracaoRepository
}

func NewConfiguracaoService(repo *repository.ConfiguracaoRepository) *ConfiguracaoService {
	return &ConfiguracaoService{repo: repo}
}

func (s *ConfiguracaoService) ListarConfiguracoes(ctx context.Context, lojaID int) (map[string]string, error) {
	configs, err := s.repo.ListarConfiguracoes(lojaID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Chave] = cfg.Valor
	}
	return result, nil
}

func (s *ConfiguracaoService) BuscarConfiguracao(ctx context.Context, lojaID int, chave string) (string, error) {
	cfg, err := s.repo.BuscarConfiguracao(lojaID, chave)
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "", nil
	}
	return cfg.Valor, nil
}

func (s *ConfiguracaoService) AtualizarConfiguracao(ctx context.Context, lojaID int, chave string, valor string) error {
	if chave == "" {
		return fmt.Errorf("chave é obrigatória")
	}
	return s.repo.AtualizarConfiguracao(lojaID, chave, valor)
}

func (s *ConfiguracaoService) AtualizarMultiplasConfiguracoes(ctx context.Context, lojaID int, configs map[string]string) error {
	for chave, valor := range configs {
		if err := s.repo.AtualizarConfiguracao(lojaID, chave, valor); err != nil {
			return fmt.Errorf("erro ao atualizar %s: %w", chave, err)
		}
	}
	return nil
}