# Plano de desenvolvimento com IA

Uma etapa por pull request, com criterios objetivos. A ordem pode ser revista por necessidade de operacao. Propostas de IA sao revisadas antes de virar regra de negocio. Este plano nao equivale a funcionalidade pronta.

## E0 — Base de cadastro e estoque

Implementada neste incremento: API Go, migracao PostgreSQL, empresas/lojas, marcas/produtos, chaves por perfil, entradas/saidas atomicas e idempotentes, testes e CI. A validacao deve ser verificada no resultado do commit. Ainda nao existe interface nem operacao comercial completa.

## E1 — Identidade de operadores e cadastros editaveis

Entregar login de pessoas, sessoes com expiracao e revogacao, vinculo de empresa/loja e trilha de alteracoes. Escolher o provedor de identidade e registrar a decisao antes de implementar. Adicionar edicao de descricao e marca/codigo com controle de concorrencia e registro do antes/depois; produtos com historico nao podem ser excluidos fisicamente.

Criterios: usuario de outra empresa nao le nem altera dados; usuario restrito a uma loja nao lanca em outra; credencial revogada/expirada falha; duas edicoes simultaneas nao se sobrescrevem silenciosamente. Cobrir as rotas de cadastro, estoque e sessao, inclusive casos negativos.

## E2 — Interface de catalogo e estoque

Entregar busca paginada, filtro de marca/loja, formulario de cadastro, consulta de saldo e historico de movimentos. Usar vermelho, preto e branco com contraste legivel. Integrar a API real e a identidade da E1; nao usar banco de dados ficticio no navegador como se fosse persistencia.

Criterios: cadastrar marca/produto e reencontrar apos recarregar; fazer uma entrada e verificar loja/produto/motivo/autor; repetir envio sem dobrar saldo; impedir permissao indevida; navegar por teclado e conferir telas estreitas com Playwright. Medir busca com base sintetica de 100 mil produtos em ambiente definido, sem prometer desempenho sem medicao.

## E3 — Catalogo tecnico de autopecas

Entregar OEMs, referencias similares, substituicoes, configuracoes de veiculo, aplicacoes, fontes/evidencias e estados confirmado/pendente. Manter SKU e codigo do fabricante separados e preservar variantes. Implementar exportacao do bloco ERP na ordem oficial: APLICACOES, REFERENCIAS SIMILARES, REFERENCIAS ORIGINAIS, DADOS COMPLEMENTARES.

Criterios: aplicacao confirmada exige fonte tecnica/OEM e configuracao; similar, variante, par LD/LE e kit/componente sao tipos separados; referencias nao criam aplicacao automaticamente. Exportacao de dados tecnicos para futura empresa exclui estoque, custos, clientes e demais dados comerciais. Guardar origem e versao dos dados importados.

## E4 — Ajustes, estornos, transferencias e importacoes

Entregar estorno relacionado ao movimento original, contagem fisica com autorizacao, transferencia com as duas pontas na mesma transacao e importacao com previa revisavel, chave de origem, relatorio de rejeicoes e retentativa idempotente.

Criterios: transferencia incompleta nao altera nenhuma ponta; duas operacoes simultaneas nao geram saldo impossivel; arquivo repetido nao duplica saldo; SKU/loja e unidade ambiguos bloqueiam somente os registros afetados e sao relatados. Saldo negativo recebido de origem e anomalia, sem ser transformado em zero ou ajustado por suposicao.

## E5 — Compras, vendas e financeiro

Definir documento e ciclo de vida antes de cada fluxo: orcamento/pedido/venda/devolucao, reserva/baixa, compras/recebimento, pagar/receber, caixa e conciliacao. Valores usam representacao decimal com precisao documentada. Regras de DMM, estoque de seguranca e compra dependem das formulas e entradas aprovadas; nao copie regras de outro repo como fatos comerciais sem revisar o contrato.

Criterios por documento: estados e permissoes explicitos, cancelamento e estorno rastreaveis, consistencia entre documento/estoque/financeiro, repeticao sem duplicidade e testes de falhas parciais. A regra de negocio deve ser escolhida antes do codigo, sem presumir como a empresa opera.

## E6 — Fiscal e piloto de operacao

Detalhar UF, regimes, tipos de documento, integrador fiscal, contingencia e requisitos de homologacao com a empresa e responsaveis tecnicos. Esse levantamento sera feito com documentacao oficial vigente; nao ha motor fiscal nesta entrega.

Criterios para piloto: ambiente de homologacao separado, migracao ensaiada com conciliacao e plano de retorno, restore de backup executado, medicao de desempenho com concorrencia representativa, teste de acesso por perfil/loja e contingencia definida. A troca do ERP atual e uma implantacao propria, posterior a esses criterios, sem parar lojas por uma troca de codigo.

## Como pedir a proxima etapa ao Copilot

Abra uma tarefa para E1 com este escopo: leia AGENTS.md e docs/ARQUITETURA.md; proponha a integracao de identidade e as mudancas de schema necessarias; implemente os criterios de E1 em uma branch; mantenha os testes de estoque existentes; execute make check test integration build; entregue PR com evidencias e limitacoes. Consulte Context7 para APIs externas quando disponivel. Nao implemente E2 a E6 nessa mesma tarefa.

## Configuracao MCP

O exemplo em `docs/copilot-mcp.json` adiciona documentacao Context7 sem chave. Copie-o para Settings → Copilot → MCP. A configuracao do servidor ainda precisa ser salva/testada nessa tela: validação de JSON nao prova conectividade nem permissao. Pode manter `{ "mcpServers": {} }` se nao quiser servicos extras; GitHub e Playwright ja sao padroes. Nao ha conexao MCP com o banco de producao.
