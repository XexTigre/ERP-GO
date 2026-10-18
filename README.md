# ERP-GO

Base de um ERP de autopecas desenvolvido por etapas com IA, Go e PostgreSQL. A primeira entrega e uma **API de cadastro e estoque**, sem interface de balcão, vendas, financeiro ou emissao fiscal.

O GitHub guarda codigo, migracoes, regras e verificacoes. A aplicacao roda em servidor proprio ou hospedagem; os dados operacionais ficam no PostgreSQL. O repositorio e publico e nao deve receber dados reais das lojas.

## O que esta implementado

- Credenciais aleatorias por empresa e perfil: `admin`, `cataloger`, `stock`, `reader`. Emissao e revogacao por CLI; o banco guarda somente o hash.
- Cadastro e consulta paginada de lojas, marcas e produtos. Produtos compartilham o catalogo entre lojas da mesma empresa.
- SKU interno separado do codigo do fabricante; codigos iguais em marcas diferentes sao permitidos. Zeros, pontuacao e sufixos sao preservados.
- Entradas e saidas de estoque por loja, em unidades inteiras `UN`, com motivo, autor e saldo resultante.
- Movimento e saldo na mesma transacao; chave de idempotencia para evitar repeticao e atualizacao atomica para impedir saidas concorrentes acima do saldo.
- Migracao versionada com checksum, testes e workflow do GitHub Actions com PostgreSQL.

Edicao de cadastros, login de pessoas, interface, OEMs/aplicacoes, estornos, transferencias, importacao e operacao fiscal sao etapas futuras descritas no [plano](docs/PLANO.md). A API inicial permite apenas cadastro e leitura dos cadastros, sem excluir ou editar historico. Nao e um ERP pronto para operar lojas.

## Executar localmente

Requisitos: Go 1.27, Docker com Compose e uma porta local 5432 livre.

1. Copie `.env.example` para `.env` e defina uma senha aleatoria em `POSTGRES_PASSWORD`. Prefira senha hexadecimal para facilitar a URL de conexao.
2. Execute `docker compose up -d db` e aguarde o banco ficar saudavel.
3. Defina `DATABASE_URL` no ambiente do terminal que executara o Go. O Go nao carrega `.env` automaticamente.

Exemplo de formato da conexao, substituindo a senha localmente:

```text
postgres://erpgo:SUA_SENHA_HEXADECIMAL@127.0.0.1:5432/erpgo?sslmode=disable
```

No Bash, use `export DATABASE_URL='URL_DA_CONEXAO'`. No PowerShell, use `$env:DATABASE_URL='URL_DA_CONEXAO'`.

```sh
go run ./cmd/erp migrate
go run ./cmd/erp bootstrap -name "Nome da sua empresa"
go run ./cmd/erp serve
```

O bootstrap cria uma empresa e mostra sua credencial administrativa uma unica vez. Guarde-a localmente; nao a coloque em issues, commits ou mensagens publicas. Cada execucao de bootstrap cria uma nova empresa. As lojas sao cadastradas pela API, sem presumir seus nomes ou quantidades.

Verifique `http://127.0.0.1:8080/healthz` e `/readyz`. Os demais endpoints exigem `Authorization: Bearer TOKEN`.

```sh
go run ./cmd/erp key -tenant ID_DA_EMPRESA -role reader
go run ./cmd/erp revoke-key -id ID_DA_CREDENCIAL
```

O compose usa usuario de administracao para desenvolvimento e porta restrita ao computador local. Antes de publicar, configure TLS, usuario de banco com privilegios minimos, identidade dos operadores, backups com restauracao testada e as verificacoes operacionais do plano. A autenticacao atual e de integracao por chave, sem login interativo.

## Trabalhar com IA no GitHub

As instrucoes estao em [AGENTS.md](AGENTS.md) e [.github/copilot-instructions.md](.github/copilot-instructions.md). Escolha uma etapa do [plano](docs/PLANO.md) e atribua uma tarefa concreta ao Copilot, com os criterios daquela etapa. Revise o pull request e os testes antes de integrar.

Para a tela **Settings → Copilot → MCP**, copie o conteudo de [docs/copilot-mcp.json](docs/copilot-mcp.json). Ele adiciona somente Context7, opcional, para consultar documentacao. GitHub e Playwright ja vem habilitados. Apenas criar o arquivo no repositorio nao configura essa tela.

Sem chave, Context7 tem limites de uso. Se precisar de chave, crie o segredo `COPILOT_MCP_CONTEXT7_API_KEY` no ambiente `copilot` e acrescente `"headers": {"Authorization": "Bearer $COPILOT_MCP_CONTEXT7_API_KEY"}` ao servidor. A chave real fica no segredo. Essa configuracao vale para o agente e a revisao na nuvem; IDE e CLI usam configuracoes proprias.

Fontes: [MCP no GitHub](https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/configure-mcp-servers), [Context7 no Copilot](https://context7.com/docs/resources/all-clients), [exemplo sem chave do Context7](https://github.com/upstash/context7/blob/master/i18n/README.pt-BR.md).

## Verificar

```sh
make check test build
```

Integracao exige um banco descartavel cujo nome termine em `_test`; os testes recusam outro nome e criam seu proprio schema. Defina `TEST_DATABASE_URL` antes de executar:

```sh
make integration
```

No Windows sem Make, execute `go vet ./...`, `go test -race -count=1 ./...`, `go test -tags=integration -race -count=1 ./...` e `go build ./cmd/erp`. O detector de corridas exige CGO e um compilador C; o workflow Linux executa essas verificacoes.

Os testes de integracao cobrem isolamento por empresa e loja, perfis, duplicidade, saldo insuficiente, retentativas simultaneas e saidas concorrentes. Usam somente dados sinteticos. Consulte os resultados do Actions do commit correspondente antes de considerar a etapa validada.

Consulte a [API](docs/API.md), a [arquitetura](docs/ARQUITETURA.md) e o [plano de desenvolvimento](docs/PLANO.md).
