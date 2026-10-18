'use strict';

// This reference preview uses only synthetic in-memory data. It never stores
// credentials, writes to the ERP API or uses browser storage as a database.
const ROUTES=[['catalog','Consulta de peças','search'],['product','Cadastro de produto','box'],['stock','Estoque por loja','store'],['movements','Movimentos','history']];
const ICONS={
 search:'<circle cx="10.8" cy="10.8" r="6.8"/><path d="m16 16 5 5"/>',
 box:'<path d="m12 3 9 5-9 5-9-5 9-5Zm-9 5v9l9 5 9-5V8M12 13v9M7.5 5.5l9 5"/>',
 store:'<path d="M3 10h18l-2-7H5l-2 7Zm1 0v11h16V10M9 21v-7h6v7M3 10c0 3 4 3 4 0 0 3 5 3 5 0 0 3 5 3 5 0 0 3 4 3 4 0"/>',
 history:'<path d="M3 11a9 9 0 1 1 2 7M3 4v7h7M12 7v5l3 2"/>',
 plus:'<path d="M12 5v14M5 12h14"/>',arrow:'<path d="M5 12h14m-6-6 6 6-6 6"/>',
 info:'<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7v1"/>',
 lock:'<rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v3"/>',
 check:'<path d="m5 12 4 4L19 6"/>',offline:'<path d="m3 3 18 18M2 8a17 17 0 0 1 4-2m4-1a17 17 0 0 1 12 3M5 12a12 12 0 0 1 5-2m5 1a12 12 0 0 1 4 1M8 16a6 6 0 0 1 8 0M12 20h.01"/>'
};
const esc=s=>String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const txt=(s,cls='',attrs='')=>`<span data-t class="${cls}" ${attrs}>${esc(s)}</span>`;
const icon=name=>`<svg data-i="${name}" aria-hidden="true" focusable="false" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${ICONS[name]||ICONS.box}</svg>`;
const tag=(s,tone='')=>`<span data-f data-component="Badge" data-variant="Tone=${tone||'neutral'}" class="tag ${tone}">${txt(s)}</span>`;
const btn=(s,{kind='',action='',route='',disabled=false,iconName=''}={})=>`<button data-f data-component="Button" data-variant="Style=${kind||'secondary'}, State=${disabled?'disabled':'default'}" class="btn ${kind}" type="button" ${action?`data-action="${action}"`:''} ${route?`data-route="${route}"`:''} ${disabled?'disabled':''}>${iconName?icon(iconName):''}${txt(s)}</button>`;
const heading=(eyebrow,title,subtitle,action='')=>`<div data-f class="page-head"><div data-f class="title-block">${txt(eyebrow,'eyebrow')}<h1 data-t class="title" tabindex="-1">${esc(title)}</h1><p data-t class="subtitle">${esc(subtitle)}</p></div>${action}</div>`;
const field=(label,value,{name,width='wide',options,type='text',hint='',readonly=false}={})=>`<div data-f class="field ${width}"><label data-t for="${name}">${esc(label)}</label>${options?`<select id="${name}" data-t>${options.map(v=>`<option${v===value?' selected':''}>${esc(v)}</option>`).join('')}</select>`:type==='textarea'?`<textarea id="${name}" data-t>${esc(value)}</textarea>`:`<input id="${name}" data-t type="${type}" value="${esc(value)}"${readonly?' readonly':''}${type==='number'?' min="1" max="1000000000" step="1" inputmode="numeric"':''}>`}${hint?txt(hint,'hint'):''}</div>`;
const PRODUCTS=[
 {sku:'DEMO-0001',code:'00.125-A',name:'Reservatório de expansão · 3 saídas',brand:'Marca A · exemplo',stock:[18,7,0,11]},
 {sku:'DEMO-0002',code:'00.125-A',name:'Reservatório de expansão · 3 saídas',brand:'Marca B · exemplo',stock:[4,2,0,3]},
 {sku:'DEMO-0003',code:'101.24 AC',name:'Filtro de ar · variante AC',brand:'Marca A · exemplo',stock:[8,0,4,2]},
 {sku:'DEMO-0004',code:'101.24 BL',name:'Filtro de ar · variante BL',brand:'Marca A · exemplo',stock:[3,2,1,0]},
 {sku:'DEMO-0005',code:'0K2A-1',name:'Terminal de direção · lado direito',brand:'Marca C · exemplo',stock:[0,0,0,0]},
 {sku:'DEMO-0006',code:'A/001-02',name:'Correia auxiliar · código completo',brand:'Marca B · exemplo',stock:[6,3,0,1]}
];
let currentRoute='catalog',selectedSKU=PRODUCTS[0].sku,branchCount=4,branches=[],activeBranch=0,moveType='entry';
let MOVEMENTS=[],demoRequests=new Map(),reviewDraft=null,pendingMovement=null,selectedMovement='',toastTimer,lastDialogFocus;
const product=()=>PRODUCTS.find(p=>p.sku===selectedSKU);
const balance=(sku,idx)=>MOVEMENTS.filter(m=>m.sku===sku&&m.branchIndex===idx).reduce((a,m)=>a+m.qty,0);
const balances=sku=>branches.map((_,i)=>balance(sku,i));
const total=sku=>balances(sku).reduce((a,b)=>a+b,0);
const normalize=s=>s.normalize('NFD').replace(/\p{M}/gu,'').toLocaleLowerCase('pt-BR').replace(/[^\p{L}\p{N}]/gu,'');
const identityKey=s=>s.trim().replace(/\s+/g,' ').toLocaleUpperCase('pt-BR');

function resetFixtures(count){
 branchCount=count;branches=Array.from({length:count},(_,i)=>i===0?'Matriz':`Loja ${i+1}`);activeBranch=0;
 MOVEMENTS=[];demoRequests=new Map();reviewDraft=null;pendingMovement=null;selectedMovement='';
 const add=(sku,origIdx,qty,reason,minute)=>{
  const idx=Math.min(origIdx,count-1),after=balance(sku,idx)+qty;
  MOVEMENTS.push({sku,branchIndex:idx,branch:branches[idx],qty,balance:after,reason,ref:`MOV-DEMO-${String(MOVEMENTS.length+1).padStart(3,'0')}`,date:`18 out · ${minute}`,type:qty>0?'Entrada':'Saída',author:'Credencial de integração'});
 };
 add('DEMO-0001',0,20,'Recebimento de exemplo','08:30');add('DEMO-0001',1,8,'Recebimento de exemplo','08:51');
 add('DEMO-0001',1,-1,'Retirada demonstrativa','09:05');add('DEMO-0001',3,11,'Recebimento de exemplo','09:18');add('DEMO-0001',0,-2,'Retirada demonstrativa','09:42');
 PRODUCTS.slice(1).forEach(p=>p.stock.forEach((q,i)=>{if(q)add(p.sku,i,q,'Carga de exemplo','08:00');}));
}
function nav(route,mobile=false){return `<nav data-f class="${mobile?'mobilebar':'nav-list'}" aria-label="${mobile?'Navegação móvel':'Navegação principal'}">${ROUTES.map(([id,label,i])=>`<a data-f data-component="Navigation" data-variant="State=${id===route?'active':'default'}" class="${mobile?'':'nav'}" href="#${id}" data-route="${id}"${id===route?' aria-current="page"':''}>${icon(i)}${txt(mobile?{catalog:'Consulta',product:'Cadastro',stock:'Estoque',movements:'Histórico'}[id]:label,'nav-label')}</a>`).join('')}</nav>`;}
function shell(route,body){return `<section data-f id="screen-${route}" data-screen="${route}" class="screen${route===currentRoute?' active':''}" aria-label="${ROUTES.find(r=>r[0]===route)[1]}"><aside data-f class="sidebar"><div data-f class="logo">${txt('GO','logo-mark')}${txt('ERP','logo-word')}</div>${txt('CATÁLOGO E ESTOQUE','side-tag')}${nav(route)}<div data-f class="side-help">${txt('Busca rápida: Ctrl K')}<br>${txt('Trocar tela: Alt 1–4')}</div><div data-f class="side-bottom"><div data-f class="side-user"><div data-f class="avatar">${txt('EX')}</div><div data-f class="col" style="gap:2px">${txt('Ambiente de exemplo','user-name')}${txt('Credencial de integração','user-role')}</div></div>${txt('V1 · referência de interface','side-version')}</div></aside><div data-f class="workspace"><header data-f class="topbar"><div data-f class="context">${icon('store')}<div data-f class="col" style="gap:2px">${txt('Empresa Demonstração','code')}${txt('Dados inteiramente sintéticos','crumb')}</div></div><div data-f class="top-end">${tag('Catálogo compartilhado','red')}${tag('UN · unidades inteiras')}</div></header><div data-f class="offline-banner" role="status">${icon('offline')}${txt('Sem conexão. Consulte os exemplos; o envio está indisponível.')}</div><main data-f class="main" id="main-${route}">${body}</main>${nav(route,true)}</div></section>`;}
function catalog(){return shell('catalog',`${heading('CATÁLOGO / CONSULTA','A peça certa começa pelo código.','Consulte a identidade completa e o saldo da sua loja.',btn('Novo produto',{kind:'primary',iconName:'plus',route:'product'}))}<div data-f class="search-block"><label class="sr-only" for="catalogSearch">Buscar por código, SKU, descrição ou marca</label><div data-f class="search-line"><div data-f class="search">${icon('search')}<input id="catalogSearch" type="search" placeholder="Código, SKU, descrição ou marca" autocomplete="off" data-t>${txt('Ctrl K','key')}</div>${btn('Buscar por código',{action:'scanner',iconName:'search'})}</div><div data-f class="filters"><div data-f class="filter-field"><label data-t for="brandFilter">Marca</label><select id="brandFilter"><option value="">Todas as marcas</option>${[...new Set(PRODUCTS.map(p=>p.brand))].map(b=>`<option>${esc(b)}</option>`).join('')}</select></div><div data-f class="filter-field"><label data-t for="storeFilter">Minha loja</label><select id="storeFilter">${branches.map((b,i)=>`<option value="${i}">${esc(b)}</option>`).join('')}</select></div><div data-f class="filter-field"><label data-t for="stockFilter">Disponibilidade</label><select id="stockFilter"><option value="">Todos os produtos</option><option value="available">Com saldo na minha loja</option></select></div></div></div><div data-f class="card catalog-desktop"><div class="table-scroll"><table class="catalog-table"><caption class="sr-only">Produtos e disponibilidade por loja</caption><thead><tr><th scope="col">PRODUTO / SKU</th><th scope="col">CÓDIGO DO FABRICANTE</th><th scope="col">MARCA</th><th scope="col" class="quantity">MINHA LOJA</th><th scope="col" class="quantity">OUTRAS</th><th scope="col" class="quantity">TOTAL</th><th scope="col"><span class="sr-only">Abrir estoque</span></th></tr></thead><tbody id="productRows"></tbody></table></div><div data-f class="table-footer">${txt('','small','id="resultCount" aria-live="polite"')}${tag('Unidade UN')}</div></div><div data-f class="mobile-result" id="mobileProducts"></div><div data-f class="empty card" id="noResults">${icon('search')}<h2 data-t class="card-title">Nenhuma peça encontrada</h2><p data-t>Tente outro código, marca ou remova os filtros.</p>${btn('Limpar busca',{action:'clear'})}</div><div data-f class="info-strip">${icon('info')}${txt('Mesmo código em marcas diferentes representa produtos distintos. Busca tolerante à pontuação não altera a identidade.','small')}</div><div data-f class="balance-summary" id="catalogSummary"></div><div data-f class="note">${txt('Aplicação por veículo, placa e GTIN ainda não disponíveis. Use o código completo, o SKU ou a descrição.')}</div>`);}
function productScreen(){return shell('product',`${heading('CATÁLOGO / NOVO PRODUTO','Identidade antes de equivalência.','SKU interno e código do fabricante têm funções distintas.',tag('Cadastro de exemplo','red'))}<div data-f class="two-col"><div data-f class="identity"><div data-f class="card pad form-stack"><h2 data-t class="card-title">Identificação comercial</h2><div data-f class="field-grid">${field('SKU INTERNO','DEMO-0001',{name:'productSku',width:'half',hint:'Único dentro da empresa.'})}${field('MARCA','Marca A · exemplo',{name:'productBrand',width:'half',options:[...new Set(PRODUCTS.map(p=>p.brand))]})}${field('CÓDIGO DO FABRICANTE','00.125-A',{name:'productCode',hint:'Preserve zeros, pontuação e sufixos.'})}${field('DESCRIÇÃO','Reservatório de expansão · 3 saídas',{name:'productDescription',type:'textarea'})}${field('UNIDADE','UN · unidade inteira',{name:'productUnit',readonly:true})}</div><div class="validation" id="productValidation" role="alert"></div></div><div data-f class="card pad form-stack"><div data-f class="block-title"><h2 data-t class="card-title">Dados técnicos</h2>${tag('Pendente de pesquisa','amber')}</div>${txt('Aplicação, similar e OEM exigem confirmação por fonte técnica. Não são inferidos do código.','small')}<div data-f class="erp-preview">${[['APLICAÇÕES','Pendente de configuração e fonte técnica.'],['REFERÊNCIAS SIMILARES','Nenhuma equivalência validada neste exemplo.'],['REFERÊNCIAS ORIGINAIS','Nenhum OEM pesquisado neste exemplo.'],['DADOS COMPLEMENTARES','Descrição ilustrativa: reservatório com 3 saídas.']].map(([a,b])=>`<div data-f class="erp-part"><strong data-t>${a}</strong>${txt(b)}</div>`).join('')}</div>${btn('Copiar modelo do bloco ERP',{action:'copy-block'})}</div></div><aside data-f class="right-col"><div data-f class="card pad form-stack"><h2 data-t class="card-title">Mesmo código, outra marca?</h2>${txt('As marcas A e B usam 00.125-A nos exemplos. São dois cadastros, dois SKUs e saldos independentes.','small')}<div data-f class="divider"></div>${tag('Marca + código completo','red')}${txt('Caixa e espaços são normalizados para a identidade. Zeros e sufixos permanecem.','small')}</div></aside></div><div data-f class="page-footer">${txt('Validação demonstrativa. O produto não será gravado.','small')}<div data-f class="row">${btn('Cancelar',{route:'catalog'})}${btn('Validar cadastro de exemplo',{kind:'primary',action:'validate-product'})}</div></div>`);}
function stock(){return shell('stock',`${heading('ESTOQUE / DISPONIBILIDADE','O saldo, loja por loja.','Confira a unidade, a loja e o motivo antes de lançar.',btn('Ver histórico',{iconName:'history',route:'movements'}))}<div data-f id="stockProduct"></div><div data-f class="branch-grid" id="branchCards"></div><div data-f class="stock-layout"><div data-f class="stock-left"><div data-f class="card"><div data-f class="pad"><h2 data-t class="card-title">Disponibilidade nas lojas</h2></div><div id="stockLines" data-f></div></div><div data-f class="card pad form-stack"><h2 data-t class="card-title">Cada movimento tem um motivo.</h2>${txt('O histórico guarda a credencial responsável e o saldo após aquela operação. A identificação de pessoas será adicionada em etapa própria.','small')}<div data-f class="row wrap">${btn('Transferir entre lojas',{disabled:true,iconName:'lock'})}${tag('Em planejamento','amber')}</div></div></div><div data-f class="stock-right"><div data-f class="card pad form-stack"><div data-f class="block-title"><h2 data-t class="card-title">Registrar movimento</h2>${tag('Simulação','red')}</div><div data-f class="segmented" role="group" aria-label="Tipo de movimento"><button data-f type="button" aria-pressed="true" data-move-type="entry">${txt('Entrada')}</button><button data-f type="button" aria-pressed="false" data-move-type="exit">${txt('Saída')}</button></div><div data-f class="field-grid">${field('LOJA',branches[0],{name:'moveBranch',width:'half',options:branches})}${field('QUANTIDADE (UN)','1',{name:'moveQty',type:'number',width:'half'})}${field('MOTIVO','Recebimento demonstrativo',{name:'moveReason'})}</div><div data-f class="stock-callout"><div data-f class="col" style="gap:5px">${txt('SALDO ATUAL DA LOJA','label')}${txt('','code','id="beforeBalance"')}</div><div data-f class="col" style="gap:5px;text-align:right">${txt('APÓS A SIMULAÇÃO','label')}${txt('','code','id="afterBalance"')}</div></div><div class="validation" id="stockValidation" role="alert"></div>${btn('Revisar movimento',{kind:'primary',action:'review-move'})}${txt('Os exemplos mudam apenas nesta sessão. Recarregar restaura a demonstração.','micro')}</div><div data-f class="note">${txt('O saldo não é editado diretamente. Estorno, contagem e transferência terão fluxos próprios.')}</div></div></div>`);}
function movements(){return shell('movements',`${heading('ESTOQUE / RAZÃO DE MOVIMENTOS','Um histórico que permanece.','Consulte a operação original e o saldo resultante.',btn('Registrar movimento',{kind:'primary',iconName:'plus',route:'stock'}))}<div data-f class="filters"><div data-f class="filter-field"><label data-t for="movementBranch">Loja do histórico</label><select id="movementBranch"><option value="">Todas as lojas</option>${branches.map(b=>`<option>${esc(b)}</option>`).join('')}</select></div><div data-f class="filter-field"><label data-t for="movementType">Tipo de movimento</label><select id="movementType"><option value="">Entradas e saídas</option><option>Entrada</option><option>Saída</option></select></div><div data-f id="historyProduct"></div></div><div data-f class="ledger"><div data-f class="ledger-left"><div data-f class="card"><div class="table-scroll"><table class="ledger-table"><caption class="sr-only">Histórico do produto selecionado</caption><thead><tr><th scope="col">DATA / OPERAÇÃO</th><th scope="col">LOJA</th><th scope="col">MOTIVO</th><th scope="col" class="quantity">QTDE.</th><th scope="col" class="quantity">SALDO APÓS</th><th scope="col">AUTORIA</th></tr></thead><tbody id="movementRows"></tbody></table></div><div data-f class="table-footer">${txt('','small','id="movementCount" aria-live="polite"')}${tag('UN')}</div></div><div data-f class="info-strip">${icon('info')}${txt('“Saldo após” pertence àquela operação e àquela loja. O saldo atual aparece na consulta de estoque.','small')}</div><div data-f class="card pad form-stack"><h2 data-t class="card-title">Conferência do exemplo</h2><div data-f class="row wrap spread" id="ledgerCheck"></div></div></div><aside data-f class="ledger-right"><div data-f class="card pad form-stack" id="movementDetail"></div></aside></div>`);}

function render(){
 document.getElementById('app').innerHTML=[catalog(),productScreen(),stock(),movements()].join('');
 drawCatalog();drawStock();drawHistory();navigate(currentRoute,false);
 if(document.body.classList.contains('is-offline'))setOffline(true);
}
function drawCatalog(){
 const q=document.getElementById('catalogSearch').value.trim().toLocaleLowerCase('pt-BR'),n=normalize(q),b=document.getElementById('brandFilter').value,only=document.getElementById('stockFilter').value;
 const hits=PRODUCTS.filter(p=>(!q||[p.sku,p.code,p.name,p.brand].some(s=>s.toLocaleLowerCase('pt-BR').includes(q)||(n&&normalize(s).includes(n))))&&(!b||p.brand===b)&&(!only||balance(p.sku,activeBranch)>0));
 document.getElementById('productRows').innerHTML=hits.map(p=>`<tr data-f data-component="ProductRow" data-variant="State=default" data-product="${p.sku}"><td><div data-f class="col" style="gap:4px">${txt(p.name,'product-name')}${txt(p.sku,'sku')}</div></td><td>${txt(p.code,'code')}</td><td>${txt(p.brand,'brand-cell')}</td><td class="quantity">${txt(balance(p.sku,activeBranch),'quantity '+(!balance(p.sku,activeBranch)?'zero':''))}</td><td class="quantity">${txt(total(p.sku)-balance(p.sku,activeBranch),'quantity')}</td><td class="quantity">${txt(total(p.sku),'quantity total')}</td><td class="action-col"><button data-f class="open-link" data-open-product="${p.sku}" aria-label="Ver estoque de ${esc(p.sku)}">${icon('arrow')}</button></td></tr>`).join('');
 document.getElementById('mobileProducts').innerHTML=hits.map(p=>`<article data-f class="card mobile-card" data-product="${p.sku}"><div data-f class="row spread"><div data-f class="col" style="gap:4px">${txt(p.name,'product-name')}${txt(p.sku,'sku')}</div><button data-f class="open-link" data-open-product="${p.sku}" aria-label="Ver estoque de ${esc(p.sku)}">${icon('arrow')}</button></div><div data-f class="row wrap">${txt(p.code,'code')}${txt(p.brand,'brand-cell')}</div><div data-f class="mobile-quantities">${[[branches[activeBranch],balance(p.sku,activeBranch)],['Outras lojas',total(p.sku)-balance(p.sku,activeBranch)],['Total',total(p.sku)]].map(([label,q])=>`<div data-f>${txt(label,'small')}${txt(`${q} UN`,'code '+(!q?'zero':''))}</div>`).join('')}</div></article>`).join('');
 document.getElementById('resultCount').textContent=`${hits.length} de ${PRODUCTS.length} produtos sintéticos`;
 document.getElementById('noResults').style.display=hits.length?'none':'flex';
 const me=PRODUCTS.reduce((s,p)=>s+balance(p.sku,activeBranch),0),all=PRODUCTS.reduce((s,p)=>s+total(p.sku),0);
 document.getElementById('catalogSummary').innerHTML=[[branches[activeBranch],me],['Outras lojas',all-me],['Todas as lojas',all]].map(([l,q])=>`<div data-f class="balance-mini">${txt(l,'mini-title')}${txt(`${q} UN`,'mini-value')}</div>`).join('');
}
function drawStock(){
 const p=product();document.getElementById('stockProduct').innerHTML=`<div data-f class="product-banner"><div data-f class="product-icon">${icon('box')}</div><div data-f class="col fill" style="gap:4px">${txt(p.name,'product-name')}<div data-f class="row wrap">${txt(p.code,'code')}${txt(`${p.brand} / SKU ${p.sku}`,'sku')}</div></div>${tag(`${total(p.sku)} UN no total`,'red')}</div>`;
 document.getElementById('branchCards').innerHTML=branches.map((b,i)=>{const q=balance(p.sku,i);return `<div data-f class="card branch-card${i===activeBranch?' current':''}" data-component="BranchCard" data-variant="State=${i===activeBranch?'current':q?'available':'empty'}"><div data-f class="branch-top">${txt(b,'branch-name')}${tag(i===activeBranch?'Minha loja':q?'Com saldo':'Sem saldo',i===activeBranch?'red':q?'green':'amber')}</div><div data-f class="row" style="gap:6px;align-items:baseline">${txt(q,'branch-value'+(!q?' zero':''))}${txt('UN','unit')}</div>${txt('Saldo de exemplo','branch-meta')}</div>`;}).join('');
 document.getElementById('stockLines').innerHTML=branches.map((b,i)=>`<div data-f class="stock-line"><div data-f class="row">${icon('store')}<div data-f class="col" style="gap:3px">${txt(b,'code')}${txt(i===activeBranch?'Loja selecionada':'Mesma empresa','sku')}</div></div>${txt(`${balance(p.sku,i)} UN`,'value'+(!balance(p.sku,i)?' zero':''))}</div>`).join('');
 updateBalancePreview();
}
function drawHistory(){
 const p=product(),b=document.getElementById('movementBranch').value,t=document.getElementById('movementType').value;
 const all=MOVEMENTS.filter(m=>m.sku===p.sku),hits=all.filter(m=>(!b||m.branch===b)&&(!t||m.type===t)).slice().reverse();
 document.getElementById('historyProduct').innerHTML=tag(`${p.sku} / ${p.code}`,'red');
 document.getElementById('movementRows').innerHTML=hits.map(m=>`<tr data-f class="movement-row${m.ref===selectedMovement?' selected':''}" data-component="MovementRow" data-variant="State=${m.ref===selectedMovement?'selected':'default'}" data-movement="${m.ref}"><td><div data-f class="col" style="gap:4px">${txt(m.date,'movement-date')}${tag(m.type,m.qty>0?'green':'red')}</div></td><td>${txt(m.branch)}</td><td>${txt(m.reason,'movement-reason')}</td><td class="movement-q ${m.qty>0?'positive':'negative'}">${txt(`${m.qty>0?'+':''}${m.qty}`)}</td><td class="movement-q">${txt(m.balance)}</td><td><div data-f class="col" style="gap:6px">${txt(m.author,'movement-author')}<button type="button" class="btn small" data-detail="${m.ref}" aria-label="Detalhe de ${m.ref}">${txt('Detalhe')}</button></div></td></tr>`).join('');
 document.getElementById('movementCount').textContent=hits.length?`${hits.length} movimentos sintéticos · mais recentes primeiro`:'Nenhum movimento com esses filtros.';
 const entry=all.filter(m=>m.qty>0).reduce((s,m)=>s+m.qty,0),exit=-all.filter(m=>m.qty<0).reduce((s,m)=>s+m.qty,0);
 document.getElementById('ledgerCheck').innerHTML=[`Entradas: ${entry} UN`,`Saídas: ${exit} UN`,`Saldo nas ${branchCount} lojas: ${total(p.sku)} UN`].map(s=>txt(s,'small')).join('');
 const m=hits.find(x=>x.ref===selectedMovement)||hits[0];
 document.getElementById('movementDetail').innerHTML=m?`<div data-f class="block-title"><h2 data-t class="card-title">Detalhe do movimento</h2>${tag(m.type,m.qty>0?'green':'red')}</div>${txt(m.ref,'code')}<div data-f class="detail-stat">${txt('QUANTIDADE / LOJA','label')}${txt(`${m.qty>0?'+':''}${m.qty} UN / ${m.branch}`,'card-title')}${txt(`Saldo após: ${m.balance} UN`,'small')}</div><div data-f class="detail-list">${[['Produto',`${p.sku} · ${p.code}`],['Marca',p.brand],['Motivo',m.reason],['Autoria',m.author],['Origem','Lançamento demonstrativo']].map(([l,v])=>`<div data-f class="detail-item"><strong data-t>${l}</strong>${txt(v)}</div>`).join('')}</div><div data-f class="divider"></div>${btn('Solicitar estorno',{disabled:true,iconName:'lock'})}${txt('Em planejamento. O movimento original será preservado.','footnote')}`:`<h2 class="card-title">Sem movimento selecionado</h2>${txt('Ajuste os filtros para consultar o histórico.','small')}`;
}
function navigate(route,focus=false){
 if(!ROUTES.some(r=>r[0]===route))route='catalog';currentRoute=route;
 document.querySelectorAll('[data-screen]').forEach(el=>el.classList.toggle('active',el.dataset.screen===route));
 document.querySelector('.skip').href=`#main-${route}`;
 if(focus)document.querySelector(`#screen-${route} h1`).focus({preventScroll:true});
 window.scrollTo({top:0});
}
function toast(message,error=false){clearTimeout(toastTimer);const n=document.getElementById('toast');n.textContent=message;n.className='toast visible'+(error?' error':'');toastTimer=setTimeout(()=>n.classList.remove('visible'),6000);}
function updateBalancePreview(){
 const idx=branches.indexOf(document.getElementById('moveBranch').value),qty=Number(document.getElementById('moveQty').value),before=balance(selectedSKU,idx),after=before+(moveType==='entry'?qty:-qty);
 document.getElementById('beforeBalance').textContent=`${before} UN`;document.getElementById('afterBalance').textContent=Number.isFinite(after)?`${after} UN`:'—';document.getElementById('afterBalance').className='code '+(after<0?'negative':'positive');
}
function invalidateReview(){reviewDraft=null;pendingMovement=null;}
function validateProduct(){
 if(document.body.classList.contains('is-offline'))return;
 const sku=document.getElementById('productSku').value,code=document.getElementById('productCode').value,brand=document.getElementById('productBrand').value,description=document.getElementById('productDescription').value,box=document.getElementById('productValidation');
 const err=![sku,code,description].every(s=>s.trim())?'Preencha SKU, marca, código e descrição.':PRODUCTS.some(p=>p.brand===brand&&identityKey(p.code)===identityKey(code))?'Marca + código já cadastrados. Confira a marca e o sufixo.':PRODUCTS.some(p=>identityKey(p.sku)===identityKey(sku))?'SKU interno já utilizado. Informe um SKU livre.':'';
 box.textContent=err;box.classList.toggle('show',!!err);if(!err)toast('Cadastro de exemplo validado. Nenhum produto foi gravado.');
}
function reviewMovement(){
 if(document.body.classList.contains('is-offline'))return;
 const branch=document.getElementById('moveBranch').value,idx=branches.indexOf(branch),qty=Number(document.getElementById('moveQty').value),reason=document.getElementById('moveReason').value.trim(),box=document.getElementById('stockValidation');
 const payload={sku:selectedSKU,idx,qty,reason,type:moveType},fingerprint=JSON.stringify(payload),isRetry=reviewDraft?.fingerprint===fingerprint&&demoRequests.has(reviewDraft.key);
 const err=idx<0?'Escolha uma loja.':!Number.isSafeInteger(qty)||qty<=0||qty>1000000000?'Informe uma quantidade inteira de 1 a 1.000.000.000.':!reason||reason.length>300?'Informe um motivo de até 300 caracteres.':!isRetry&&moveType==='exit'&&qty>balance(selectedSKU,idx)?'Saldo insuficiente nesta loja. A saída não pode ser confirmada.':'';
 box.textContent=err;box.classList.toggle('show',!!err);if(err)return;
 // Allocate once at review. Reopening or retrying keeps the key. An edit
 // invalidates this draft, so a different payload always receives a new key.
 if(!reviewDraft||reviewDraft.fingerprint!==fingerprint)reviewDraft={...payload,fingerprint,key:crypto.randomUUID()};
 pendingMovement=reviewDraft;
 document.getElementById('dialogBody').innerHTML=[['Produto',selectedSKU],['Loja',branch],[moveType==='entry'?'Entrada':'Saída',`${qty} UN`],['Motivo',reason]].map(([l,v])=>`<div class="summary-line"><span>${esc(l)}</span><strong>${esc(v)}</strong></div>`).join('');
 lastDialogFocus=document.activeElement;document.getElementById('dialog').showModal();document.getElementById('cancelDialog').focus();
}
function closeDialog(){document.getElementById('dialog').close();pendingMovement=null;lastDialogFocus?.focus();}
function confirmMovement(){
 if(!pendingMovement||document.body.classList.contains('is-offline'))return;
 const m=pendingMovement,button=document.getElementById('confirmDialog');button.disabled=true;
 try{
  const original=demoRequests.get(m.key);
  if(original){closeDialog();toast(`Simulação já confirmada: ${original.ref}. Saldo preservado.`);return;}
  const after=balance(m.sku,m.idx)+(m.type==='entry'?m.qty:-m.qty);
  if(after<0||after>9000000000000){closeDialog();toast('Saldo indisponível para esta operação.',true);return;}
  const v={sku:m.sku,branchIndex:m.idx,branch:branches[m.idx],qty:m.type==='entry'?m.qty:-m.qty,balance:after,reason:m.reason,ref:`MOV-DEMO-${String(MOVEMENTS.length+1).padStart(3,'0')}`,date:'18 out · simulação',type:m.type==='entry'?'Entrada':'Saída',author:'Credencial de integração'};
  MOVEMENTS.push(v);demoRequests.set(m.key,v);selectedMovement=v.ref;closeDialog();drawCatalog();drawStock();drawHistory();toast('Simulação confirmada. Consulta, estoque e histórico atualizados nesta sessão.');
 }finally{button.disabled=false;}
}
function setOffline(value){document.body.classList.toggle('is-offline',value);document.querySelectorAll('[data-action="review-move"],[data-action="validate-product"]').forEach(b=>b.disabled=value);if(value&&document.getElementById('dialog').open)closeDialog();}
function clearFilters(){document.getElementById('catalogSearch').value='';document.getElementById('brandFilter').value='';document.getElementById('stockFilter').value='';drawCatalog();}
document.addEventListener('click',e=>{
 const route=e.target.closest('[data-route]');if(route){e.preventDefault();history.replaceState(null,'',`#${route.dataset.route}`);navigate(route.dataset.route,true);return;}
 const open=e.target.closest('[data-open-product]');if(open){selectedSKU=open.dataset.openProduct;selectedMovement='';invalidateReview();drawStock();drawHistory();navigate('stock',true);return;}
 const detail=e.target.closest('[data-detail]');if(detail){selectedMovement=detail.dataset.detail;drawHistory();document.getElementById('movementDetail').setAttribute('tabindex','-1');document.getElementById('movementDetail').focus({preventScroll:true});return;}
 const type=e.target.closest('[data-move-type]');if(type){moveType=type.dataset.moveType;document.querySelectorAll('[data-move-type]').forEach(b=>b.setAttribute('aria-pressed',String(b===type)));invalidateReview();updateBalancePreview();return;}
 const action=e.target.closest('[data-action]');if(!action||action.disabled)return;
 switch(action.dataset.action){case'clear':clearFilters();document.getElementById('catalogSearch').focus();break;case'scanner':document.getElementById('catalogSearch').focus();toast('Busque pelo código do fabricante ou SKU. GTIN ainda não está disponível.');break;case'validate-product':validateProduct();break;case'review-move':reviewMovement();break;case'copy-block':{const text='APLICAÇÕES\nPendente de configuração + fonte técnica.\n\nREFERÊNCIAS SIMILARES\nNenhuma equivalência validada neste exemplo.\n\nREFERÊNCIAS ORIGINAIS\nNenhum OEM pesquisado neste exemplo.\n\nDADOS COMPLEMENTARES\nDescrição ilustrativa: reservatório com 3 saídas.';if(!navigator.clipboard){toast('Cópia indisponível neste navegador.',true);break;}navigator.clipboard.writeText(text).then(()=>toast('Modelo copiado. Dados técnicos seguem pendentes.')).catch(()=>toast('Não foi possível copiar o modelo.',true));break;}}
});
document.addEventListener('input',e=>{if(e.target.id==='catalogSearch')drawCatalog();if(['moveBranch','moveQty','moveReason'].includes(e.target.id)){invalidateReview();updateBalancePreview();}});
document.addEventListener('change',e=>{
 if(['brandFilter','stockFilter'].includes(e.target.id))drawCatalog();
 if(e.target.id==='storeFilter'){activeBranch=Number(e.target.value);document.getElementById('moveBranch').value=branches[activeBranch];invalidateReview();drawCatalog();drawStock();}
 if(['movementBranch','movementType'].includes(e.target.id))drawHistory();
 if(e.target.id==='offlineToggle')setOffline(e.target.checked);
 if(e.target.id==='branchCount'){resetFixtures(Number(e.target.value));render();toast(`Exemplo reiniciado com ${branchCount} lojas.`);}
});
document.getElementById('cancelDialog').onclick=closeDialog;document.getElementById('confirmDialog').onclick=confirmMovement;
document.getElementById('dialog').addEventListener('cancel',()=>{pendingMovement=null;});
document.getElementById('dialog').addEventListener('keydown',e=>{
 if(e.key!=='Tab')return;
 const first=document.getElementById('cancelDialog'),last=document.getElementById('confirmDialog');
 if(e.shiftKey&&document.activeElement===first){e.preventDefault();last.focus();}
 else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first.focus();}
});
document.addEventListener('keydown',e=>{
 if(document.getElementById('dialog').open)return;
 if(e.ctrlKey&&e.key.toLowerCase()==='k'){e.preventDefault();navigate('catalog');document.getElementById('catalogSearch').focus();}
 if(e.altKey&&['1','2','3','4'].includes(e.key)){e.preventDefault();navigate(ROUTES[Number(e.key)-1][0],true);}
});
resetFixtures(4);currentRoute=ROUTES.some(r=>r[0]===location.hash.slice(1))?location.hash.slice(1):'catalog';render();
window.GO_ERP={navigate,PRODUCTS,ICONS,esc,txt,icon,tag,btn,heading,get MOVEMENTS(){return MOVEMENTS;},get reviewKey(){return reviewDraft?.key;},get branchCount(){return branchCount;}};
