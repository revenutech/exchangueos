package config

import (
	"strings"
	"testing"
)

// TestDBConfig_Redacted_NeverLeaksPassword é a prova de regressão para o
// vazamento de senha em claro descoberto no harmonyos-revenu (RedactDSN,
// revenu-common/db/cockroach.go, commit 7aab03b): url.Parse NÃO retorna erro
// para uma DSN no formato keyword/value do libpq (host=x password=y ...) —
// devolve a string quase intacta e u.User fica nil, então a checagem
// "if u.User != nil" nunca redige a senha.
//
// SENHA_DE_TESTE é uma senha falsa usada só para este teste — nunca um segredo
// real.
func TestDBConfig_Redacted_NeverLeaksPassword(t *testing.T) {
	const senhaDeTeste = "SENHA_DE_TESTE"

	cases := map[string]string{
		"url_form": "postgresql://usuario:" + senhaDeTeste + "@host:26257/db?sslmode=verify-full",
		"keyword_value_form": "host=x port=26257 user=usuario password=" + senhaDeTeste + " dbname=db",
	}

	for name, dsn := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := DBConfig{DSN: dsn}
			got := cfg.Redacted()
			if strings.Contains(got, senhaDeTeste) {
				t.Fatalf("Redacted() vazou a senha em claro para %s: %q", name, got)
			}
		})
	}
}
