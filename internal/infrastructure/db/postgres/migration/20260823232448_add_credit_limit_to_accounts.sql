-- +goose Up

-- Limite de crédito da account, em centavos. NULL significa "sem limite" —
-- estado válido apenas para tipos diferentes de credit_card.
ALTER TABLE accounts ADD COLUMN credit_limit BIGINT NULL;

-- Rede de segurança para as invariantes do domínio (a validação primária é da
-- entidade Account): credit_card tem limite positivo e dentro do teto de
-- R$ 10 bilhões; os demais tipos não têm limite.
--
-- Sem backfill de propósito: se existir alguma account credit_card sem limite,
-- este ALTER falha ao validar as linhas atuais — preferimos a migration falhar
-- a inventar um limite de crédito.
ALTER TABLE accounts ADD CONSTRAINT accounts_credit_limit_type_check
    CHECK (
        (type = 'credit_card'
            AND credit_limit IS NOT NULL
            AND credit_limit > 0
            AND credit_limit <= 1000000000000)
        OR
        (type <> 'credit_card' AND credit_limit IS NULL)
    );

-- Nenhum índice: não há filtro, ordenação ou join por credit_limit — a
-- consolidação lê a coluna junto da própria linha da account.

-- +goose Down

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_credit_limit_type_check;
ALTER TABLE accounts DROP COLUMN IF EXISTS credit_limit;
