#!/usr/bin/env python3
"""Import the pinned Yingce video protocol engine and declarative catalog."""
import argparse, json, shutil, zipfile
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--source',type=Path,required=True);p.add_argument('--commit',required=True);a=p.parse_args()
root=Path(__file__).resolve().parents[1]
engine=root/'backend/internal/pkg/yingceprotocol';engine.mkdir(parents=True,exist_ok=True)
for source in (a.source/'backend/internal/protocol').glob('*.go'):
 if source.name.endswith('_test.go'):continue
 text=source.read_text().replace('package protocol','package yingceprotocol',1)
 if source.name=='manifest.go':
  text=text.replace('path := strings.ReplaceAll(manifestString(evaluatedPath), "{{taskId}}", url.PathEscape(taskID))','escapedTaskID := strings.ReplaceAll(url.PathEscape(taskID), "%2F", "/")\n path := strings.ReplaceAll(manifestString(evaluatedPath), "{{taskId}}", escapedTaskID)')
 (engine/source.name).write_text(text)
shutil.copytree(a.source/'backend/internal/protocol/docs',engine/'docs',dirs_exist_ok=True)
shutil.copyfile(a.source/'LICENSE',root/'LICENSE.yingce')
entries=[]
for source in sorted((a.source/'plugin-packages').glob('*.yingce-plugin')):
 with zipfile.ZipFile(source) as z: manifest=json.loads(z.read('manifest.json'))
 for provider in manifest.get('contributes',{}).get('providers',[]):
  if 'video' not in provider.get('capabilities',[]):continue
  frozen=dict(manifest);frozen['contributes']=dict(manifest['contributes']);frozen['contributes']['providers']=[provider]
  entries.append({'id':provider['id'],'name':provider['label'],'vendor':manifest.get('author',''),'version':manifest['version'],'package':source.name,'manifest':frozen})
catalog={'source_repository':'ddcat-ai/open-ai-canvas','source_commit':a.commit,'protocols':entries}
(root/'video-protocol-catalog.json').write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n')
(root/'backend/internal/pkg/videoprotocol/catalog_gen.json').write_text(json.dumps(catalog,ensure_ascii=False,separators=(',',':'))+'\n')
(root/'backend/internal/service/video_yingce_url_gen.json').write_text((root/'video-protocol-url-defaults.json').read_text())
print('Imported',len(entries),'video protocols from',a.commit)
