-- ExchangeOS — 000013: contraparte + lado no Quote
--
-- O agregado Quote não carregava contraparte nem lado, então o handler de
-- quote.accepted não conseguia bookar um trade sem inventar os dois. Sem estas
-- colunas o auto-booking fica desligado.
--
-- side decide o preço aplicado (BUY paga o ask, SELL recebe o bid) e qual
-- contraparte fica em cada perna do trade resultante.

BEGIN;

ALTER TABLE quotes ADD COLUMN IF NOT EXISTS requester_bic STRING(11);
ALTER TABLE quotes ADD COLUMN IF NOT EXISTS provider_bic  STRING(11);
ALTER TABLE quotes ADD COLUMN IF NOT EXISTS side          STRING(4);

-- Backfill: as linhas existentes foram gravadas sem contraparte nem lado. Não
-- há valor correto a inferir — um lado chutado inverte as pernas do trade e
-- troca bid por ask. Ficam NULL e são rejeitadas na leitura pelo domínio.
--
-- Pelo mesmo motivo as colunas NÃO são NOT NULL: exigir isso quebraria o
-- deploy sobre uma tabela populada. A obrigatoriedade é imposta por
-- domain.NewQuote, e as linhas legadas não são reidratáveis.

ALTER TABLE quotes ADD CONSTRAINT chk_quotes_side
    CHECK (side IS NULL OR side IN ('BUY','SELL'));

ALTER TABLE quotes ADD CONSTRAINT chk_quotes_bic_len
    CHECK (
        (requester_bic IS NULL OR length(requester_bic) IN (8, 11)) AND
        (provider_bic  IS NULL OR length(provider_bic)  IN (8, 11))
    );

-- Contraparte não pode negociar consigo mesma.
ALTER TABLE quotes ADD CONSTRAINT chk_quotes_distinct_parties
    CHECK (requester_bic IS NULL OR provider_bic IS NULL OR requester_bic <> provider_bic);

-- Busca por contraparte é o acesso natural para exposição e triagem de sanções.
CREATE INDEX IF NOT EXISTS idx_quotes_requester ON quotes (tenant_id, requester_bic, valid_to DESC);

COMMIT;
