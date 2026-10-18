# Segurança

Relate vulnerabilidades pelo [canal privado de advisories](https://github.com/XexTigre/ERP-GO/security/advisories/new). Não inclua credenciais, dados comerciais, documentos fiscais reais ou exportações em issues públicas.

Informe versão/commit, reprodução mínima com dados sintéticos, impacto e resultado esperado. Não há prazo de resposta garantido nesta fase.

A V1 está em desenvolvimento. Antes de produção, valide identidade de pessoas, permissões por loja, TLS, rotação/expiração de chaves, limites de requisição, restauração e monitoração. Use `erp_app` para runtime e `erp_migrator` para administração. RLS complementa os filtros por tenant; a credencial de manutenção não deve servir a API.
