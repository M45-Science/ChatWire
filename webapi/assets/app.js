'use strict';
const $ = selector => document.querySelector(selector);
let me, servers=[], selected='', page='Overview', dirty=false, refreshTimer;
const content=$('#content'),notice=$('#notice'),selector=$('#server');
function node(tag,text,cls){const el=document.createElement(tag);if(text!==undefined)el.textContent=text;if(cls)el.className=cls;return el;}
function notify(text,error=false){notice.textContent=text;notice.className=error?'error':'';}
async function api(path,options={}){const viewKey=selected+'|'+page;const headers={...options.headers};if(options.body&&!(options.body instanceof FormData))headers['Content-Type']='application/json';if(options.method&&options.method!=='GET')headers['X-CSRF-Token']=me?.csrf_token||'';const response=await fetch(path,{...options,headers});const result=await response.json();if(!response.ok)throw new Error(result.error?.message||`Request failed (${response.status})`);if((!options.method||options.method==='GET')&&viewKey!==selected+'|'+page){const e=new Error('View changed.');e.name='AbortError';throw e;}return result;}
function base(){return '/api/v1/servers/'+encodeURIComponent(selected);}
function panel(title){const p=node('section',undefined,'panel');if(title)p.append(node('h2',title));content.append(p);return p;}
function button(text,fn,cls='quiet'){const b=node('button',text,cls);b.type='button';b.addEventListener('click',()=>Promise.resolve(fn()).catch(e=>notify(e.message,true)));return b;}
// Match fact.LevelToString; the API continues to accept numeric levels.
const playerLevels=new Map([[-255,'Deleted'],[-1,'Banned'],[0,'New'],[1,'Member'],[2,'Regular'],[3,'Veteran'],[255,'Moderator']]);
function playerLevelName(value){
 if(value===null||value===undefined||value==='')return 'Unknown';
 const level=Number(value);
 if(!Number.isInteger(level))return 'Unknown';
 if(level<=-254)return 'Deleted';
 if(level>=255)return 'Moderator';
 return playerLevels.get(level)||'Invalid';
}
function dataTable(rows,keys){const wrap=node('div',undefined,'table-scroll'),table=node('table'),head=node('tr');keys.forEach(k=>head.append(node('th',k.replaceAll('_',' '))));const thead=node('thead');thead.append(head);table.append(thead);const body=node('tbody');for(const row of rows){const tr=node('tr');for(const k of keys)tr.append(node('td',k.toLowerCase()==='level'?playerLevelName(row[k]):String(row[k]??'')));body.append(tr);}table.append(body);wrap.append(table);return wrap;}
async function loadServers(){servers=await api('/api/v1/servers');selector.replaceChildren(new Option('All servers',''));for(const s of servers)selector.add(new Option(s.label+(s.available?'':' · offline'),s.id));selector.value=selected;}
function navigation(){const names=['Overview','Settings','Actions','Players','Maps & saves','Mods','Console','Firewall','Activity'];if(me.actor.admin)names.push('Shared settings','Host settings');$('#nav').replaceChildren(...names.map(name=>button(name,async()=>{if(dirty&&!confirm('Discard unsaved changes?'))return;dirty=false;page=name;await render();},name===page?'active':'')));$('#nav').querySelector('button.active')?.setAttribute('aria-current','page');}
async function render(){navigation();$('#title').textContent=page;$('#scope').textContent=selected?(servers.find(s=>s.id===selected)?.label||selected):'ALL LOCAL INSTANCES';content.replaceChildren();notify('');if(page!=='Overview'&&page!=='Shared settings'&&page!=='Activity'&&page!=='Host settings'&&page!=='Firewall'&&!selected){panel().append(node('p','Choose a local instance to view these controls.'));return;}
 try{switch(page){case 'Overview':await overview();break;case 'Settings':await settings(false);break;case 'Shared settings':await settings(true);break;case 'Actions':await actions();break;case 'Players':await players();break;case 'Maps & saves':await saves();break;case 'Mods':await mods();break;case 'Console':await consolePage();break;case 'Activity':await activity();break;case 'Host settings':await hostSettings();break;case 'Firewall':await firewallPage();break;}}catch(e){if(e.name!=='AbortError')notify(e.message,true);}}
async function overview(){
 await loadServers();
 const cards=node('div',undefined,'cards');content.append(cards);
 for(const s of servers){
  if(selected&&s.id!==selected)continue;
  const card=node('section',undefined,'card');
  const phase=s.available?s.status.lifecycle.Phase:'Offline';
  card.dataset.state=!s.available?'offline':phase==='running'?'running':phase==='stopped'?'stopped':'busy';
  card.append(node('h2',s.label),node('p',phase,'status'));
  if(s.available){
   const metrics=node('dl',undefined,'server-metrics');
   for(const [label,value]of [['Players',s.status.players],['Autostart',s.status.autostart?'On':'Off']]){
    const metric=node('div');metric.append(node('dt',label),node('dd',String(value)));metrics.append(metric);
   }
   card.append(metrics,node('p',`Factorio ${s.status.version} · ${s.status.map||'No map loaded'}`,'meta'));
   const c=s.status.configuration;
   if(c&&(c.local_desired_revision!==c.local_runtime_revision||c.global_desired_revision!==c.global_runtime_revision))card.append(node('p','Saved configuration differs from runtime.','meta'));
   if(s.status.lifecycle.PendingAction)card.append(node('p','Pending: '+s.status.lifecycle.PendingAction,'meta'));
  }else card.append(node('p','ChatWire is unavailable. Factorio state is unknown.','meta'));
  card.append(button('Manage server',async()=>{selected=s.id;selector.value=selected;page='Actions';await render();}));
  cards.append(card);
 }
 if(!selected)await bulkControls(content);
}
async function settings(global){const root=global?'/api/v1/settings/global':base()+'/settings';const [schema,current]=await Promise.all([api(root+'/schema'),api(root)]);const p=panel(global?'Shared configuration':'Server configuration');p.append(node('p',current.application));const form=node('form'),inputs=[];for(const f of schema){if(f.read_only||f.admin&&!me.actor.admin)continue;const row=node('label',undefined,'field'),label=node('span',f.label||f.key);label.append(node('small',f.effect.replaceAll('_',' ')+(f.secret?' · stored value hidden':'')));let input;if(f.choices?.length){input=node('select');f.choices.forEach(v=>input.add(new Option(v,v)));input.value=current.values[f.key]??'';}else{input=node('input');input.type=f.secret?'password':f.type==='bool'?'checkbox':(f.type==='int'||f.type==='float32')?'number':'text';if(input.type==='checkbox')input.checked=!!current.values[f.key];else if(!f.secret)input.value=current.values[f.key]??'';if(f.secret)input.placeholder=current.values[f.key]?.configured?'Configured — leave blank to keep':'Not configured';if(f.min!==undefined)input.min=f.min;if(f.max!==undefined)input.max=f.max;if(f.type==='float32')input.step='any';}input.addEventListener('input',()=>{dirty=true;});row.append(label,input);form.append(row);let clearInput;if(f.secret){const clearLabel=node('label',undefined,'field');clearLabel.append(node('span','Clear stored '+(f.label||f.key)));clearInput=node('input');clearInput.type='checkbox';clearInput.addEventListener('change',()=>dirty=true);clearLabel.append(clearInput);form.append(clearLabel);}inputs.push({f,input,clearInput});}
 const save=node('button','Save settings');save.type='submit';form.append(save);form.addEventListener('submit',async e=>{e.preventDefault();save.disabled=true;try{const patch={};for(const {f,input,clearInput}of inputs){let v=input.type==='checkbox'?input.checked:input.type==='number'?Number(input.value):input.value;if(f.secret){if(clearInput.checked)v='';else if(!v)continue;}if(JSON.stringify(v)!==JSON.stringify(current.values[f.key]))patch[f.key]=v;}if(!Object.keys(patch).length){notify('No changes to save.');return;}if(patch['Options.SoftModOptions.OneLife']===true&&!confirm('Enable one-life permadeath? This cannot be undone on the current map.'))return;await api(root,{method:'PATCH',headers:{'If-Match':'"'+current.revision+'"'},body:JSON.stringify(patch)});dirty=false;await render();notify('Saved. Reload configuration or use the indicated restart to apply.');}catch(err){notify(err.message,true);}finally{save.disabled=false;}});p.append(form);}
const fields={
 'factorio-restart':[['when_empty','Wait until empty','checkbox']], 'chatwire-restart':[['when_empty','Wait until empty','checkbox'],['force','Force reboot','checkbox']],
 'map-load':[['save_id','Save ID','text']], 'map-archive':[['save_id','Save ID','text']], 'map-exchange':[['exchange_string','Map exchange string','textarea']],
 'rcon':[['command','Console command','textarea']], 'mods-edit':[['operation','Operation','select',['add','remove','enable','disable','version']],['mods','Mod names, comma separated','text'],['version','Version or auto','text']],
 'player-level':[['player','Player name','text'],['level','Level','select',[...playerLevels.keys()]]], 'ip-ban':[['ip','Public IPv4 address','text']], 'ip-unban':[['ip','Public IPv4 address','text']],
 'upload-apply':[['upload_ids','Staged upload IDs, comma separated','text'],['start','Start after applying','checkbox']]
};
function actionForm(container,name,preset={},submitAction=runAction){const form=node('form');const entries=[...(fields[name]||[]),['reason','Reason','text']];const inputs=[];for(const [key,label,type,choices]of entries){const row=node('label',undefined,'field');row.append(node('span',label));const input=node(type==='textarea'?'textarea':type==='select'?'select':'input');if(type==='select')choices.forEach(v=>input.add(new Option(key==='level'?playerLevelName(v):v,v)));else if(type!=='textarea')input.type=type;if(preset[key]!==undefined)input.value=preset[key];row.append(input);form.append(row);inputs.push({key,type,input});}const submit=node('button','Preview action');submit.type='submit';form.append(submit);form.addEventListener('submit',async e=>{e.preventDefault();const payload={};for(const {key,type,input}of inputs){let v=type==='checkbox'?input.checked:input.value;if(key==='mods'||key==='upload_ids')v=v.split(',').map(x=>x.trim()).filter(Boolean);if(key==='level')v=Number(v);payload[key]=v;}try{await submitAction(name,payload);}catch(error){notify(error.message,true);}});container.append(form);}
async function actions(){const list=await api(base()+'/actions');const p=panel('Choose an action'),select=node('select');for(const a of list)select.add(new Option(a.name,a.name));p.append(select);const detail=node('div');p.append(detail);function show(){detail.replaceChildren(node('p',list.find(a=>a.name===select.value).description));actionForm(detail,select.value);}select.addEventListener('change',show);show();}
async function runAction(name,payload){const targetId=selected;const target=base(),preview=await api(target+'/actions/'+name+'/preview',{method:'POST',body:JSON.stringify(payload)});$('#confirm-text').textContent=`${name} on ${servers.find(s=>s.id===targetId)?.label||targetId}. ${preview.players} players connected.`;$('#confirm-detail').textContent=JSON.stringify(name==='player-level'?{...payload,level:playerLevelName(payload.level)}:payload,null,2);const dialog=$('#confirm');dialog.showModal();const confirmed=await new Promise(resolve=>{const cleanup=()=>{dialog.removeEventListener('cancel',cancel);$('#cancel').onclick=null;$('#execute').onclick=null;};const cancel=()=>{cleanup();dialog.close();resolve(false);};$('#cancel').onclick=cancel;dialog.addEventListener('cancel',cancel);$('#execute').onclick=()=>{cleanup();dialog.close();resolve(true);};});if(!confirmed)return;const job=await api(target+'/actions/'+name,{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({...payload,confirmation_token:preview.confirmation_token})});selected=targetId;selector.value=selected;page='Activity';await render();notify(`Accepted ${job.action}.`);}
async function saves(){
 const root=base(),p=panel('Saved maps');
 async function appendPage(cursor=''){
  const result=await api(root+'/saves?cursor='+encodeURIComponent(cursor));
  for(const save of result.items){const row=node('div',undefined,'field');const label=node('span',save.name);label.append(node('small',`${(save.size/1048576).toFixed(1)} MiB`));const controls=node('div',undefined,'buttons');const link=node('a','Download');link.href=root+'/saves/'+encodeURIComponent(save.id)+'/download';controls.append(link,button('Load',()=>runAction('map-load',{save_id:save.id})),button('Archive',()=>runAction('map-archive',{save_id:save.id})));row.append(label,controls);p.append(row);}
  if(result.next_cursor){const more=button('Load more saves',async()=>{more.remove();await appendPage(result.next_cursor);});p.append(more);}
 }
 await appendPage();
 const up=panel('Stage an upload'),kind=node('select');[['save','Save game (.zip)'],['mod-list','Mod list (.json)'],['mod-settings','Mod settings (.dat)']].forEach(([v,t])=>kind.add(new Option(t,v)));const file=node('input');file.type='file';
 up.append(kind,file,button('Upload',async()=>{if(!file.files[0])throw new Error('Choose a file.');const body=new FormData();body.append(kind.value,file.files[0]);const u=await api(root+'/uploads',{method:'POST',body});const detail=node('div');detail.append(node('p',`Staged ${u.name}. Applying will stop Factorio.`));actionForm(detail,'upload-apply',{upload_ids:u.id});detail.append(button('Discard upload',async()=>{await api(root+'/uploads/'+encodeURIComponent(u.id),{method:'DELETE'});detail.remove();}));up.append(detail);}));
}
async function mods(){const result=await api(base()+'/mods');panel('Mod list').append(dataTable(result.mods||result.Mods||[],['name','enabled']));const edit=panel('Edit mods');actionForm(edit,'mods-edit');const history=await api(base()+'/mods/history');panel('History').append(node('pre',history.history));}
async function players(){
 const online=await api(base()+'/players');panel('Online in this instance').append(dataTable(online.items,['Name','Level','AFK']));
 const listPanel=panel('Shared player database'),search=node('input');search.type='search';search.placeholder='Search player names';search.setAttribute('aria-label','Search player names');const rows=node('div');
 let query='';async function appendPage(cursor=''){const result=await api('/api/v1/players?q='+encodeURIComponent(query)+'&cursor='+encodeURIComponent(cursor));rows.append(dataTable(result.items,['name','level','minutes','ban_reason']));if(result.next_cursor){const more=button('Load more players',async()=>{more.remove();await appendPage(result.next_cursor);});rows.append(more);}}
 listPanel.append(search,button('Search',async()=>{query=search.value;rows.replaceChildren();await appendPage();}),rows);await appendPage();
 const p=panel('Change player level');const list=await api(base()+'/actions');if(list.some(a=>a.name==='player-level'))actionForm(p,'player-level');else p.append(node('p','Select the primary instance to change shared player levels.'));
}
async function consolePage(){actionForm(panel('Remote console'),'rcon');const result=await api(base()+'/logs');panel('Recent log output').append(node('pre',result.text));}
async function activity(){const result=selected?{[selected]:await api(base()+'/jobs')}:await api('/api/v1/jobs');for(const [id,jobs]of Object.entries(result)){const p=panel(servers.find(s=>s.id===id)?.label||id);if(!Array.isArray(jobs)){p.append(node('p','Instance unavailable.'));continue;}for(const job of jobs){const details=node('details');details.append(node('summary',`${job.action} · ${job.state} · ${new Date(job.created_at).toLocaleString()}`));details.append(node('pre',JSON.stringify({id:job.id,actor:job.actor.name,state:job.state,error:job.error,result:job.result},null,2)));p.append(details);}}}
selector.addEventListener('change',async()=>{if(dirty&&!confirm('Discard unsaved changes?')){selector.value=selected;return;}dirty=false;selected=selector.value;await render();});$('#refresh').addEventListener('click',()=>{if(!dirty)render();});$('#logout').addEventListener('click',async()=>{try{await api('/auth/logout',{method:'POST'});location.assign('/login');}catch(e){notify(e.message,true);}});window.addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue='';}});
(async()=>{try{me=await api('/api/v1/me');$('#identity').textContent=me.actor.name+(me.actor.admin?' · Admin':' · Moderator');await loadServers();await render();const events=new EventSource('/api/v1/events');events.addEventListener('resync_required',()=>{clearTimeout(refreshTimer);if(!dirty&&(page==='Overview'||page==='Activity'))refreshTimer=setTimeout(()=>render(),250);});events.onerror=()=>{notify('Live updates disconnected. Refresh to verify access and server state.');};}catch(e){content.replaceChildren();panel('Sign in through Discord').append(node('p','Run /web in Discord and open the private link to access your server controls.'));notify(e.message,true);}})();

let lastActivity=0;for(const event of ['pointerdown','keydown'])document.addEventListener(event,()=>{if(me&&Date.now()-lastActivity>60000){lastActivity=Date.now();api('/auth/activity',{method:'POST'}).catch(e=>notify(e.message,true));}});

async function bulkControls(container){
 const p=node('section',undefined,'panel');p.append(node('h2','Run across selected servers'));const action=node('select');action.add(new Option('Console command','rcon'));action.add(new Option('Reload saved configuration','config-reload'));const command=node('textarea');command.placeholder='Factorio console command';p.append(action,command);action.addEventListener('change',()=>{command.hidden=action.value!=='rcon';});const targets=[];for(const s of servers){const label=node('label',undefined,'field'),input=node('input');input.type='checkbox';input.disabled=!s.available;label.append(node('span',s.label+(s.available?'':' · offline')),input);p.append(label);targets.push({id:s.id,input});}p.append(button('Preview selected servers',async()=>{const payload={action:action.value,servers:targets.filter(x=>x.input.checked).map(x=>x.id),parameters:action.value==='rcon'?{command:command.value}:{}};const preview=await api('/api/v1/batches/preview',{method:'POST',body:JSON.stringify(payload)});if(!confirm(`Run ${payload.action} on ${preview.servers.join(', ')}?`))return;await api('/api/v1/batches',{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({...payload,confirmation_token:preview.confirmation_token})});page='Activity';await render();}));container.append(p);
}
async function hostSettings(){const current=await api('/api/v1/host/settings');const p=panel('Web service configuration');p.append(node('p','Changes are saved for the next web-service restart. Credential fields contain protected file paths, not secret values.'));const form=node('form'),text=node('textarea');text.rows=24;text.value=JSON.stringify(current.values,null,2);text.addEventListener('input',()=>dirty=true);const submit=node('button','Save host configuration');submit.type='submit';form.append(text,submit);form.addEventListener('submit',async e=>{e.preventDefault();try{await api('/api/v1/host/settings',{method:'PATCH',headers:{'If-Match':'"'+current.revision+'"'},body:JSON.stringify(JSON.parse(text.value))});dirty=false;notify('Saved. A web-service restart is required to apply these changes.');}catch(err){notify(err.message,true);}});p.append(form);}

async function runHostAction(action,payload){
 const root='/api/v1/host/actions/'+action;await api(root+'/preview',{method:'POST',body:JSON.stringify(payload)}).then(async preview=>{
 if(!confirm(`${action} ${payload.ip} on this host? This affects all local servers.`))return;
 await api(root,{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({...payload,confirmation_token:preview.confirmation_token})});selected='';selector.value='';page='Activity';await render();
 });
}
async function firewallPage(){
 const ips=await api('/api/v1/host/ip-bans');const p=panel('Host firewall deny rules');if(!ips.length)p.append(node('p','No public IPv4 deny rules.'));for(const ip of ips){const row=node('div',undefined,'field');row.append(node('span',ip),button('Remove deny rule',()=>runHostAction('ip-unban',{ip})));p.append(row);}actionForm(panel('Deny a public IPv4 address'),'ip-ban',{},runHostAction);
}
