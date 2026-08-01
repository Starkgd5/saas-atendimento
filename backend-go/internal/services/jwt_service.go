package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey  string
	expiration time.Duration
}

type Claims struct {
	UserID   int    `json:"user_id"`
	LojaID   int    `json:"loja_id"`
	Role     string `json:"role"`
	Email    string `json:"email,omitempty"`
	Nome     string `json:"nome,omitempty"`
	jwt.RegisteredClaims
}

// ============================================
// CONSTRUTOR
// ============================================

func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey:  secretKey,
		expiration: 24 * time.Hour, // padrão: 24 horas
	}
}

// NewJWTServiceWithExpiration cria um serviço JWT com expiração personalizada
func NewJWTServiceWithExpiration(secretKey string, expiration time.Duration) *JWTService {
	return &JWTService{
		secretKey:  secretKey,
		expiration: expiration,
	}
}

// ============================================
// GERAÇÃO DE TOKENS
// ============================================

// GenerateToken gera um novo token JWT
func (s *JWTService) GenerateToken(userID, lojaID int, role string) (string, error) {
	return s.GenerateTokenWithDetails(userID, lojaID, role, "", "")
}

// GenerateTokenWithDetails gera um token com informações adicionais
func (s *JWTService) GenerateTokenWithDetails(userID, lojaID int, role, email, nome string) (string, error) {
	claims := &Claims{
		UserID: userID,
		LojaID: lojaID,
		Role:   role,
		Email:  email,
		Nome:   nome,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", userID),
			ID:        fmt.Sprintf("%d-%d", userID, time.Now().UnixNano()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// GenerateRefreshToken gera um token de refresh com maior duração
func (s *JWTService) GenerateRefreshToken(userID, lojaID int, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		LojaID: lojaID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 dias
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ============================================
// VALIDAÇÃO DE TOKENS
// ============================================

// ValidateToken valida e extrai claims do token
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inválido: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("erro ao validar token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token inválido")
}

// ValidateTokenWithOptions valida token com opções adicionais
func (s *JWTService) ValidateTokenWithOptions(tokenString string, options ...jwt.ParserOption) (*Claims, error) {
	parser := jwt.NewParser(options...)
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inválido: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("erro ao validar token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token inválido")
}

// ============================================
// VERIFICAÇÕES ESPECÍFICAS
// ============================================

// VerifyToken verifica se o token é válido e retorna o usuário ID
func (s *JWTService) VerifyToken(tokenString string) (int, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// VerifyTokenWithRole verifica token e valida role
func (s *JWTService) VerifyTokenWithRole(tokenString string, allowedRoles []string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	for _, role := range allowedRoles {
		if claims.Role == role {
			return claims, nil
		}
	}

	return nil, fmt.Errorf("role não autorizada: %s", claims.Role)
}

// IsTokenExpired verifica se o token expirou
func (s *JWTService) IsTokenExpired(tokenString string) (bool, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return true, err
	}
	return claims.ExpiresAt.Time.Before(time.Now()), nil
}

// GetExpirationTime retorna o tempo de expiração do token
func (s *JWTService) GetExpirationTime(tokenString string) (time.Time, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}

// ============================================
// CONFIGURAÇÃO
// ============================================

// SetExpiration define o tempo de expiração padrão
func (s *JWTService) SetExpiration(expiration time.Duration) {
	s.expiration = expiration
}

// GetExpiration retorna o tempo de expiração padrão
func (s *JWTService) GetExpiration() time.Duration {
	return s.expiration
}

// ============================================
// UTILITÁRIOS
// ============================================

// ExtractUserID extrai apenas o UserID do token sem validação completa
func (s *JWTService) ExtractUserID(tokenString string) (int, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(*Claims); ok {
		return claims.UserID, nil
	}
	return 0, fmt.Errorf("não foi possível extrair UserID")
}

// ExtractClaimsRaw extrai claims sem validação (apenas parse)
func (s *JWTService) ExtractClaimsRaw(tokenString string) (*Claims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok {
		return claims, nil
	}
	return nil, fmt.Errorf("não foi possível extrair claims")
}