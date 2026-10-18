import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {createServer} from 'node:http';
import {chromium} from 'playwright';
import axe from 'axe-core';

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const staticRoot=path.join(root,'internal/web/static');
const server=createServer(async(req,res)=>{
 try{
  const filename=new URL(req.url,'http://localhost').pathname;
  const allowed={'/':'index.html','/index.html':'index.html','/styles.css':'styles.css','/app.js':'app.js'};
  if(!allowed[filename]){res.writeHead(404);res.end();return;}
  const type={html:'text/html',css:'text/css',js:'text/javascript'}[allowed[filename].split('.').pop()];
  res.writeHead(200,{'Content-Type':`${type}; charset=utf-8`});res.end(await fs.readFile(path.join(staticRoot,allowed[filename])));
 }catch{res.writeHead(500);res.end();}
});
await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
const browser=await chromium.launch({headless:true,...(process.env.GO_ERP_CHROME?{executablePath:process.env.GO_ERP_CHROME,args:['--no-sandbox']}: {})});
const page=await browser.newPage({viewport:{width:1488,height:1100}});
const report={date:new Date().toISOString(),checks:[],axe:[],contrasts:[],limits:['Protótipo sintético em memória','Importador não executado dentro do Figma']};
const errors=[];page.on('pageerror',e=>errors.push(e.message));
const check=(value,label)=>{assert.ok(value,label);report.checks.push(label);};
const route=async r=>page.evaluate(r=>window.GO_ERP.navigate(r),r);
try{
 await page.goto(`http://127.0.0.1:${server.address().port}`);
 await page.waitForSelector('#screen-catalog.active');
 check(await page.locator('#productRows tr').count()===6,'Seis produtos sintéticos na consulta');
 await page.fill('#catalogSearch','00125A');check(await page.locator('#productRows tr').count()===2,'Busca sem pontuação mantém marcas separadas');
 await page.selectOption('#brandFilter',{label:'Marca A · exemplo'});check(await page.locator('#productRows tr').count()===1,'Filtro de marca identifica o produto');
 await page.fill('#catalogSearch','inexistente');check(await page.locator('#noResults').isVisible(),'Estado vazio visível');
 await page.click('#noResults [data-action=clear]');
 await page.keyboard.press('Control+k');check(await page.locator('#catalogSearch').evaluate(e=>e===document.activeElement),'Atalho Ctrl K leva o foco à busca');
 await route('product');await page.click('[data-action=validate-product]');
 check((await page.locator('#productValidation').innerText()).includes('Marca + código'),'Duplicidade usa a identidade comercial');
 await page.fill('#productSku','DEMO-0099');await page.fill('#productCode','00.125-C');await page.click('[data-action=validate-product]');
 check(!await page.locator('#productValidation').isVisible(),'Sufixo diferente não é fundido');
 await route('stock');await page.click('[data-move-type=exit]');await page.fill('#moveQty','19');await page.click('[data-action=review-move]');
 check((await page.locator('#stockValidation').innerText()).includes('Saldo insuficiente'),'Saída acima do saldo bloqueada');
 await page.fill('#moveQty','1.5');await page.click('[data-action=review-move]');check((await page.locator('#stockValidation').innerText()).includes('inteira'),'Quantidade fracionada bloqueada');
 await page.fill('#moveQty','2');await page.click('[data-action=review-move]');
 const firstKey=await page.evaluate(()=>window.GO_ERP.reviewKey);check(!!firstKey,'Chave criada ao abrir a revisão');
 await page.keyboard.press('Shift+Tab');check(await page.locator('#confirmDialog').evaluate(e=>e===document.activeElement),'Foco fica dentro do diálogo');
 await page.keyboard.press('Escape');check(!await page.locator('#dialog').isVisible(),'Escape fecha o diálogo');
 await page.click('[data-action=review-move]');check(await page.evaluate(()=>window.GO_ERP.reviewKey)===firstKey,'Reabrir revisão reutiliza a chave');
 await page.click('#confirmDialog');
 const countAfter=await page.evaluate(()=>window.GO_ERP.MOVEMENTS.length);
 check((await page.locator('#beforeBalance').innerText())==='16 UN','Simulação atualiza o saldo');
 await page.click('[data-action=review-move]');await page.click('#confirmDialog');
 check(await page.evaluate(()=>window.GO_ERP.MOVEMENTS.length)===countAfter,'Reenvio não duplica o movimento');
 check((await page.locator('#beforeBalance').innerText())==='16 UN','Reenvio preserva o saldo');
 await page.fill('#moveQty','3');await page.click('[data-action=review-move]');
 check(await page.evaluate(()=>window.GO_ERP.reviewKey)!==firstKey,'Edição cria outra chave');await page.keyboard.press('Escape');
 await page.check('#offlineToggle');check(await page.locator('[data-action=review-move]').isDisabled(),'Sem conexão bloqueia o envio');await page.uncheck('#offlineToggle');
 await route('movements');check((await page.locator('#ledgerCheck').innerText()).includes('34 UN'),'Histórico reflete o saldo simulado');
 await route('catalog');check((await page.locator('#productRows tr').first().innerText()).includes('34'),'Consulta reflete o saldo simulado');

 for(const count of [1,2,4,12]){
  await page.selectOption('#branchCount',String(count));await route('stock');
  check(await page.locator('#branchCards .branch-card').count()===count,`${count} lojas com detalhe individual`);
  check((await page.locator('#stockProduct').innerText()).includes('36 UN'),`${count} lojas preservam o saldo do conjunto sintético`);
 }
 await page.selectOption('#branchCount','4');
 await fs.mkdir(path.join(root,'test-results'),{recursive:true});
 await page.addScriptTag({content:axe.source});
 for(const width of [1440,1280,1024,834,390,320]){
  await page.setViewportSize({width,height:1100});
  for(const r of ['catalog','product','stock','movements']){
   await route(r);
   check(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`${r}: página sem rolagem lateral em ${width}px`);
   if([1440,1024,390].includes(width))await page.screenshot({path:path.join(root,`test-results/${r}-${width}.png`),fullPage:true});
   if([1440,390].includes(width)){
    const result=await page.evaluate(async()=>await axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21a','wcag21aa','wcag22aa']}}));
    const violations=result.violations.map(v=>({id:v.id,impact:v.impact,help:v.help,nodes:v.nodes.map(n=>n.target)}));
    report.axe.push({route:r,width,violations});assert.deepEqual(violations,[],`${r} ${width}: axe violations ${JSON.stringify(violations)}`);
   }
  }
 }
 await route('catalog');
 check(await page.locator('.screen.active .mobilebar').isVisible(),'Navegação móvel visível');
 const bottom=await page.locator('.screen.active .main').evaluate(e=>parseFloat(getComputedStyle(e).paddingBottom));
 const bar=await page.locator('.screen.active .mobilebar').evaluate(e=>e.getBoundingClientRect().height);
 check(bottom>bar,'Conteúdo reserva espaço para a barra inferior');
 await page.setViewportSize({width:1440,height:1100});
 const colors=await page.evaluate(()=>{const s=getComputedStyle(document.documentElement);return Object.fromEntries(['ink','surface','muted','bg','control','primary','red','sidebar','sidebar-muted','focus','focus-dark','green','green-soft','amber','amber-soft','danger','danger-soft'].map(k=>[k,s.getPropertyValue('--'+k).trim()]));});
 const luminance=hex=>{const [r,g,b]=hex.slice(1).match(/../g).map(h=>parseInt(h,16)/255).map(c=>c<=.04045?c/12.92:((c+.055)/1.055)**2.4);return .2126*r+.7152*g+.0722*b;};
 for(const [a,b,min] of [['ink','surface',4.5],['muted','surface',4.5],['muted','bg',4.5],['control','surface',3],['control','bg',3],['primary','surface',4.5],['red','surface',4.5],['focus','surface',3],['focus-dark','sidebar',3],['sidebar-muted','sidebar',4.5],['green','green-soft',4.5],['amber','amber-soft',4.5],['danger','danger-soft',4.5]]){
  const l1=luminance(colors[a]),l2=luminance(colors[b]),ratio=(Math.max(l1,l2)+.05)/(Math.min(l1,l2)+.05);
  check(ratio>=min,`Contraste ${a}/${b}: ${ratio.toFixed(2)} >= ${min}`);report.contrasts.push({a,b,ratio,min});
 }
 check(await page.locator('input:not([type=checkbox]),select,textarea').evaluateAll(es=>es.every(e=>document.querySelector(`label[for="${e.id}"]`))),'Todos os campos têm label associado');
 check(await page.locator('.sidebar [aria-current=page]').count()===4,'Navegação usa aria-current');
 check(await page.locator('.sidebar [aria-pressed]').count()===0,'Navegação não usa aria-pressed');
 check(errors.length===0,'Nenhum erro JavaScript');
 console.log(`${report.checks.length} verificações; ${report.axe.length} auditorias axe sem violações.`);
}finally{
 await fs.mkdir(path.join(root,'test-results'),{recursive:true});await fs.writeFile(path.join(root,'test-results/v1-report.json'),JSON.stringify(report,null,2));
 await browser.close();await new Promise(resolve=>server.close(resolve));
}
