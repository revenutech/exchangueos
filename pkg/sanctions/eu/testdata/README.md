# EU FSF fixtures — synthetic

`eu-fsf-sintetico.csv` follows the layout of the EU Financial Sanctions Files consolidated list (CSV, `;`-separated,
UTF-8 with BOM, CRLF), with a subset of the official columns. Every name, reference number and programme is
invented; the countries use the ISO 3166 user-assigned codes `XA`/`XB`. Never replace it with the real list
(owner decision 2026-11-10). `golden.json` is the expected `[]sanctions.Entry`; regenerate with
`go test ./pkg/sanctions/eu -run TestParse_Golden -update`.
