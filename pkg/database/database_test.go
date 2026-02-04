// pkg/database/database_test.go
package database

import (
	"testing"
)

func TestDatabaseConnection(t *testing.T) {
	// Adapter ces valeurs à votre configuration
	db, err := NewDB("localhost", "3306", "ludo_user", "LudoPass2024!", "ludo_king")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	t.Log("✅ Database connection successful!")
}

func TestCreateUser(t *testing.T) {
	db, err := NewDB("localhost", "3306", "ludo_user", "LudoPass2024!", "ludo_king")
	if err != nil {
		t.Skip("Database not available")
	}
	defer db.Close()

	// Créer un utilisateur de test
	username := "test_user_" + randomString(8)
	email := username + "@test.com"
	passwordHash := "hashed_password"

	user, err := db.CreateUser(username, email, passwordHash)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Username != username {
		t.Errorf("Expected username %s, got %s", username, user.Username)
	}

	t.Logf("✅ User created successfully: %s (ID: %d)", user.Username, user.ID)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
