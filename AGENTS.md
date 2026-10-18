# Instrucoes do projeto ERP-GO

## Objetivo

Construir um ERP de autopecas por incrementos revisaveis, com catalogo aproveitavel, estoque por loja e isolamento entre empresas. Use a estrutura existente e prefira codigo simples. O repositório e publico: exemplos devem ser sinteticos.

## Rotina

1. Leia o README, a arquitetura e os criterios da etapa escolhida em `docs/PLANO.md`.
2. Implemente uma etapa com escopo definido. Registre decisoes que alterem regras de negocio.
3. Execute `make check test integration build`; documente qualquer verificacao que nao foi possivel executar.
4. Abra um pull request com comportamento, validacao e limites. Nao declare o ERP pronto por passar no build.

## Regras de dados

- IDs internos sao estaveis; SKU e codigo do fabricante sao campos distintos. A identidade comercial nesta etapa e empresa + marca + codigo completo.
- Preserve zeros, pontos, barras e sufixos. Normalizacao somente de caixa e espacos. Marcas diferentes podem usar o mesmo codigo.
- Empresas compartilham o catalogo entre suas lojas; saldos e movimentacoes pertencem a uma loja. Empresas diferentes tem dados separados. Reuso entre empresas sera por fluxo explicito de publicacao/importacao tecnica, nunca por acesso cruzado a dados comerciais.
- OEM, referencia similar, substituicao e aplicacao devem ser relacoes distintas quando implementados. Aplicacao confirmada exige evidencia tecnica e configuracao; a IA nao e fonte de compatibilidade.
- O bloco ERP futuro segue: APLICACOES, REFERENCIAS SIMILARES, REFERENCIAS ORIGINAIS, DADOS COMPLEMENTARES. Hipoteses ficam pendentes.
- Nesta etapa o estoque usa somente unidades inteiras UN. Nao use ponto flutuante para quantidades nem valores monetarios. Fracoes e embalagens exigem uma decisao e migracao posteriores.
- Grave movimento e saldo na mesma transacao. Repeticao da mesma chave deve retornar o resultado original; conteudo diferente com a mesma chave gera conflito.
- Nunca edite saldo diretamente pela API nem apague historico para corrigir uma operacao. Estornos, ajustes e transferencias ainda precisam de contratos proprios.
- Cada consulta e mutacao deve aplicar a empresa obtida da credencial; nunca aceite `tenant_id` enviado pelo cliente. FKs compostas preservam a empresa nas relacoes.
- Testes de mutacao devem cobrir credencial ausente, perfil sem permissao e empresa errada. Estoque exige teste de concorrencia em PostgreSQL real.

## Segredos e operacao

Nao versione tokens, `.env`, XMLs de notas reais, exportacoes, PDFs operacionais ou dados de clientes. O bootstrap gera credencial aleatoria e guarda somente SHA-256. Nao registre cabecalhos Authorization. Configure TLS e usuario de banco com privilegios minimos antes de expor a API. O compose e para desenvolvimento local.

Nao configure MCP com acesso a producao para fazer desenvolvimento. `docs/copilot-mcp.json` e um exemplo opcional a copiar nas configuracoes do GitHub, nao um arquivo carregado automaticamente.
