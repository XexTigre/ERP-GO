# API inicial

Base local: `http://127.0.0.1:8080`. Rotas `/v1/` exigem `Authorization: Bearer TOKEN`. Escritas exigem `Content-Type: application/json`. Nao ha cookies, CORS ou credenciais padroes.

| Metodo | Caminho | Acesso | Comportamento |
| --- | --- | --- | --- |
| GET | /healthz | Publico | Processo ativo |
| GET | /readyz | Publico | Conectividade com banco |
| GET | /v1/brands | Autenticado | Lista marcas |
| POST | /v1/brands | Admin/cataloger | Cadastra marca |
| GET | /v1/branches | Autenticado | Lista lojas |
| POST | /v1/branches | Admin | Cadastra loja |
| GET | /v1/products | Autenticado | Busca por SKU, codigo ou descricao |
| POST | /v1/products | Admin/cataloger | Cadastra produto |
| GET | /v1/products/{id} | Autenticado | Consulta produto da empresa |
| GET | /v1/balances?branch_id=ID | Autenticado | Saldos movimentados na loja |
| POST | /v1/movements | Admin/stock | Entrada ou saida |
| GET | /v1/movements/{id} | Autenticado | Consulta uma movimentacao |

Listas recebem `limit` entre 1 e 100 (padrao 30) e `after` com o UUID retornado em `next_cursor`. A resposta tem `items` e, quando ha outra pagina, `next_cursor`. A busca de produtos recebe `q`, ate 120 caracteres. `%` e `_` sao caracteres literais, nao curingas. A busca nao inclui OEM ou similares nesta etapa.

Corpo de cadastro de marca ou loja:

```json
{"name": "Nome escolhido"}
```

Corpo de produto, usando o ID de uma marca ja cadastrada:

```json
{
  "brand_id": "UUID_DA_MARCA",
  "sku": "SKU_INTERNO",
  "manufacturer_code": "CODIGO_COMPLETO",
  "description": "DESCRICAO_EXATA"
}
```

Para movimentar, use `Idempotency-Key` com 16 a 100 caracteres ASCII (letras, numeros, `_` ou `-`). Use uma nova chave para uma nova operacao e a mesma chave para repetir uma operacao cujo resultado nao foi recebido.

```json
{
  "branch_id": "UUID_DA_LOJA",
  "product_id": "UUID_DO_PRODUTO",
  "kind": "entrada",
  "quantity": 10,
  "reason": "Motivo da movimentacao"
}
```

`kind`: `entrada` ou `saida`. `quantity`: inteiro positivo, de 1 a 1.000.000.000; unidade fixa `UN`. Saida acima do saldo e recusada. Nao use entrada/saida para simular estorno, transferencia, ajuste de contagem ou importacao de saldo sem implementar as regras dessas operacoes.

Criacao retorna 201. Repeticao de movimento retorna 200 com o mesmo ID e `balance_after` historico da operacao original, mesmo que outros movimentos ja tenham acontecido. Reuso de chave com outro conteudo retorna 409. A chave so pode ser reaproveitada depois de uma operacao recusada, pois nao houve gravacao.

Erros: 400 para entrada invalida, 401 para credencial invalida/revogada, 403 para perfil sem permissao, 404 para registro inexistente ou de outra empresa, 409 para duplicidade/conflito/saldo insuficiente, 415 para tipo de conteudo errado. Falhas inesperadas nao devolvem detalhes do banco ao cliente.

Nao ha atualizacao ou exclusao de cadastro, lista de historico por filtros, chave por filial, login de pessoas, reservas, valores monetarios ou emissao fiscal nesta versao.
