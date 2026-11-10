package submission

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestPayloadNeverLeaksBody(t *testing.T) {
	t.Parallel()
	const cpf = "123.456.789-09"
	body := []byte("<DEC><Declarante><CPF>" + cpf + "</CPF><Nome>Joao da Silva</Nome></Declarante></DEC>")
	p := Payload{Kind: KindDEC, ReferenceID: "rep-42", Body: body}

	var outs []string
	for _, f := range []string{"%v", "%+v", "%s"} {
		outs = append(outs, fmt.Sprintf(f, p))
	}
	var text, js bytes.Buffer
	slog.New(slog.NewTextHandler(&text, nil)).Info("submit", "p", p)
	slog.New(slog.NewJSONHandler(&js, nil)).Info("submit", "p", p)
	outs = append(outs, text.String(), js.String())

	for i, out := range outs {
		if strings.Contains(out, cpf) || strings.Contains(out, "Joao") || strings.Contains(out, "<DEC>") {
			t.Errorf("output %d leaks the body: %s", i, out)
		}
		if !strings.Contains(out, "DEC") || !strings.Contains(out, "rep-42") || !strings.Contains(out, fmt.Sprint(len(body))) {
			t.Errorf("output %d lost kind, reference or body length: %s", i, out)
		}
	}
}
