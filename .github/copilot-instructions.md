# ERP-GO

Leia `AGENTS.md`, `docs/ARQUITETURA.md` e a tarefa escolhida em `docs/PLANO.md` antes de implementar.

Go + PostgreSQL, monolito modular. O primeiro incremento e uma API; a interface web e uma etapa posterior. Nao substitua a stack nem amplie uma tarefa sem necessidade comprovada.

Use Context7, quando configurado, para conferir a API e a versao de dependencias. Se o MCP estiver indisponivel, use a documentacao oficial; nao invente chamadas. GitHub e Playwright MCP ja sao padroes do Copilot no GitHub. A configuracao desta tela nao configura IDE nem CLI.

Cada tarefa deve entregar comportamento executavel, testes relevantes, comandos de verificacao e limitacoes objetivas no pull request. Nao transforme instrucao de IA em mecanismo de seguranca: autenticacao, isolamento e saldo devem ser garantidos no codigo e no banco.

Execute `make check test integration build`. Os testes de integracao exigem `TEST_DATABASE_URL` em banco exclusivo terminado em `_test`; nunca use dados ou bancos de producao. Nao declare integracao testada quando ela foi pulada.
