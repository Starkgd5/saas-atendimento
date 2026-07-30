package repository

import (
	"database/sql"
	"testing"

	_"github.com/go-sql-driver/mysql"
	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Usar banco de teste
	db, err := sql.Open("mysql", "root:root123@tcp(localhost:3306)/saas_atendimento_test?parseTime=true")
	if err != nil {
		t.Fatal(err)
	}

	return db, func() {
		db.Close()
	}
}

func TestClienteRepository_CriarCliente(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewClienteRepository(db)

	cliente := &models.Cliente{
		LojaID:   1,
		Nome:     "Teste Silva",
		Telefone: "11999999999",
		Email:    "teste@email.com",
	}

	err := repo.CriarCliente(cliente)
	assert.NoError(t, err)
	assert.NotZero(t, cliente.ID)
}
